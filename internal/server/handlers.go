package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"log/slog"

	"helixops/internal/analyzer"
	"helixops/internal/config"
	"helixops/internal/db"
	"helixops/internal/models"
	"helixops/internal/orchestrator"
	"helixops/internal/output"
	"helixops/internal/postmortem"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	cfg          *config.Config
	orchestrator *orchestrator.Orchestrator
	analyzer     *analyzer.Analyzer
	generator    *postmortem.Generator
	mdReporter   *output.MarkdownReporter
	slackSender  *output.SlackSender
	database     *db.DB
}

// NewHandler constructs a Handler struct with the necessary dependencies injected.
func NewHandler(cfg *config.Config, orch *orchestrator.Orchestrator, anlz *analyzer.Analyzer, gen *postmortem.Generator, md *output.MarkdownReporter, slack *output.SlackSender, database *db.DB) *Handler {
	return &Handler{
		cfg:          cfg,
		orchestrator: orch,
		analyzer:     anlz,
		generator:    gen,
		mdReporter:   md,
		slackSender:  slack,
		database:     database,
	}
}

// RegisterRoutes maps REST API paths to their corresponding HTTP handler methods on the provided router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/webhook", h.HandleWebhook)
	r.Get("/health", h.HandleHealth)
	r.Get("/ready", h.HandleReady)

	r.Get("/postmortems", h.HandleListPostmortems)
	r.Get("/postmortems/{id}", h.HandleGetPostmortem)
}

// HandleWebhook parses incoming HTTP POST payloads from Prometheus Alertmanager.
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size (max 1MB)
	maxBodySize := int64(1 << 20) // 1MB
	if r.ContentLength > maxBodySize {
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Read request body
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodySize))
	if err != nil {
		slog.Error("request.read_failed", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse AlertManager webhook payload
	var alertPayload models.AlertManagerPayload
	if err := json.Unmarshal(body, &alertPayload); err != nil {
		slog.Error("request.parse_failed", "error", err)
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	// Validate alerts
	if len(alertPayload.Alerts) == 0 {
		slog.Warn("webhook.empty_payload")
		http.Error(w, "No alerts in payload", http.StatusBadRequest)
		return
	}

	// Validate each alert has required fields
	for i, alert := range alertPayload.Alerts {
		if alert.Labels == nil {
			slog.Warn("alert.invalid", "index", i, "reason", "missing labels")
			alertPayload.Alerts = append(alertPayload.Alerts[:i], alertPayload.Alerts[i+1:]...)
			continue
		}
		if alert.Labels["alertname"] == "" {
			slog.Warn("alert.invalid", "index", i, "reason", "missing alertname")
			alertPayload.Alerts = append(alertPayload.Alerts[:i], alertPayload.Alerts[i+1:]...)
			continue
		}
	}

	// Re-check after filtering invalid alerts
	if len(alertPayload.Alerts) == 0 {
		http.Error(w, "No valid alerts in payload", http.StatusBadRequest)
		return
	}

	slog.Info("webhook.received", "alerts", len(alertPayload.Alerts), "receiver", alertPayload.Receiver)

	// Process alerts asynchronously
	go h.processAlerts(alertPayload)

	// Acknowledge immediately
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": fmt.Sprintf("Processing %d alerts", len(alertPayload.Alerts)),
	})
}

// processAlerts iterates through webhook payloads and asynchronously orchestrates RCA analysis or postmortem generation.
func (h *Handler) processAlerts(payload models.AlertManagerPayload) {
	for _, alert := range payload.Alerts {
		serviceName := extractServiceName(alert.Labels)
		if serviceName == "" {
			slog.Warn("alert.skip", "alert", alert.Labels["alertname"], "reason", "missing service_name")
			continue
		}

		if alert.Status == "resolved" {
			slog.Info("alert.resolved_processing", "alert", alert.Labels["alertname"], "service", serviceName)
			if h.generator == nil || h.orchestrator == nil {
				continue
			}

			// Prepare context mapping back to incident start for full postmortem view
			ctx, err := h.orchestrator.PrepareContext(context.Background(), serviceName, alert.StartsAt)
			if err != nil {
				slog.Error("postmortem.context_prepare_failed", "service", serviceName, "error", err)
				continue
			}

			// Map Alert Info
			ctx.Alert = models.AlertInfo{
				Name:      alert.Labels["alertname"],
				Severity:  alert.Labels["severity"],
				Summary:   alert.GetAnnotation("summary"),
				Labels:    alert.Labels,
				StartedAt: alert.StartsAt,
			}

			pm, err := h.generator.Generate(context.Background(), ctx)
			if err != nil {
				slog.Error("postmortem.generate_failed", "service", serviceName, "error", err)
				continue
			}

			slog.Info("postmortem.generated", "postmortem_id", pm.ID, "service", serviceName)

			// Resolve incident in database if available
			if h.database != nil {
				if err := h.database.ResolveIncident(pm.ID, pm.RootCause, pm.Markdown); err != nil {
					slog.Error("db.resolve_incident_failed", "error", err)
				} else {
					slog.Info("db.resolved_incident", "incident_id", pm.ID)
				}
			}

			if h.mdReporter != nil {
				if err := h.mdReporter.SendPostmortem(pm); err != nil {
					slog.Error("postmortem.save_markdown_failed", "error", err)
				}
			}
			continue
		}

		if alert.Status != "firing" {
			continue
		}

		slog.Info("alert.processing", "alert", alert.Labels["alertname"], "service", serviceName)

		// Guard against nil dependencies (for tests)
		if h.orchestrator == nil || h.analyzer == nil {
			slog.Warn("alert.skip", "reason", "missing orchestrator or analyzer")
			continue
		}

		// Create analysis context with metrics, logs, commits, and traces
		ctx, err := h.orchestrator.PrepareContext(context.Background(), serviceName, alert.StartsAt)
		if err != nil {
			slog.Error("context.prepare_failed", "service", serviceName, "error", err)
			continue
		}

		// Map alert info to context
		ctx.Alert = models.AlertInfo{
			Name:      alert.Labels["alertname"],
			Severity:  alert.Labels["severity"],
			Summary:   alert.GetAnnotation("summary"),
			Labels:    alert.Labels,
			StartedAt: alert.StartsAt,
		}

		// Analyze with full context (metrics, commits, traces)
		result, err := h.analyzer.AnalyzeWithContext(context.Background(), ctx)
		if err != nil {
			slog.Error("analysis.failed", "service", serviceName, "error", err)
			continue
		}

		slog.Info("analysis.complete", "service", serviceName, "summary", result.Summary)

		// Store incident in database if available
		if h.database != nil && result != nil {
			incident := &db.Incident{
				ID:          result.ID,
				ServiceName: serviceName,
				AlertName:   alert.Labels["alertname"],
				Severity:    alert.Labels["severity"],
				StartedAt:   alert.StartsAt,
			}
			if err := h.database.CreateIncident(incident); err != nil {
				slog.Error("db.create_incident_failed", "error", err)
			} else {
				slog.Info("db.created_incident", "incident_id", result.ID)
			}
		}

		// Send to output channels (Slack and Markdown)
		if h.slackSender != nil {
			if err := h.slackSender.SendAnalysis(result); err != nil {
				slog.Error("output.slack_failed", "error", err)
			} else {
				slog.Info("output.slack_sent", "service", serviceName)
			}
		}

		if h.mdReporter != nil {
			if err := h.mdReporter.Report(result); err != nil {
				slog.Error("output.markdown_failed", "error", err)
			}
		}

		// Persist analysis result (including any data gaps) into DB.analysis_results
		if h.database != nil {
			// Ensure incident exists
			incident := &db.Incident{
				ID:          result.ID,
				ServiceName: serviceName,
				AlertName:   alert.Labels["alertname"],
				Severity:    alert.Labels["severity"],
				StartedAt:   alert.StartsAt,
			}
			if err := h.database.CreateIncident(incident); err != nil {
				slog.Error("db.create_incident_failed", "error", err)
			} else {
				slog.Info("db.created_incident", "incident_id", result.ID)
			}

			// Build payload including errors
			payload := struct {
				Result *models.AnalysisResult `json:"result"`
				Errors map[string]string      `json:"errors,omitempty"`
			}{
				Result: result,
				Errors: ctx.Errors,
			}

			buf, _ := json.Marshal(payload)
			if err := h.database.CreateAnalysisResult(result.ID, "llm_analysis", string(buf)); err != nil {
				slog.Error("db.save_analysis_result_failed", "error", err)
			}
		}
	}
}

// extractServiceName attempts to identify the impacted service by scanning common metric label keys.
func extractServiceName(labels map[string]string) string {
	// Try common label names
	if name, ok := labels["service_name"]; ok {
		return name
	}
	if name, ok := labels["service"]; ok {
		return name
	}
	if name, ok := labels["job"]; ok {
		return name
	}
	return ""
}

// HandleHealth returns health status
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleReady returns readiness status
func (h *Handler) HandleReady(w http.ResponseWriter, r *http.Request) {
	// Check if orchestrator is ready
	if h.orchestrator != nil {
		ready := h.orchestrator.HealthCheck(r.Context())
		if !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "not ready",
				"reason": "orchestrator not properly initialized",
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
	})
}

// HandleListPostmortems lists generated postmortems
func (h *Handler) HandleListPostmortems(w http.ResponseWriter, r *http.Request) {
	if h.database == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Database not configured",
			"data":    []string{},
		})
		return
	}

	incidents, err := h.database.ListIncidents("resolved")
	if err != nil {
		slog.Error("db.list_incidents_failed", "error", err)
		http.Error(w, "Failed to retrieve incidents", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Retrieved resolved incidents",
		"data":    incidents,
	})
}

// HandleGetPostmortem fetches a single postmortem
func (h *Handler) HandleGetPostmortem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if h.database == nil {
		http.Error(w, "Database not configured", http.StatusNotFound)
		return
	}

	incident, err := h.database.GetIncident(id)
	if err != nil {
		slog.Error("db.get_incident_failed", "error", err)
		http.Error(w, "Failed to retrieve incident", http.StatusInternalServerError)
		return
	}

	if incident == nil {
		http.Error(w, "Incident not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           incident.ID,
		"service_name": incident.ServiceName,
		"alert_name":   incident.AlertName,
		"severity":     incident.Severity,
		"started_at":   incident.StartedAt,
		"resolved_at":  incident.ResolvedAt,
		"root_cause":   incident.RootCause,
		"status":       incident.Status,
	})
}
