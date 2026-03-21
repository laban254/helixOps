package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

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
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse AlertManager webhook payload
	var alertPayload models.AlertManagerPayload
	if err := json.Unmarshal(body, &alertPayload); err != nil {
		log.Printf("Failed to parse webhook payload: %v", err)
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	// Validate alerts
	if len(alertPayload.Alerts) == 0 {
		log.Printf("No alerts in payload")
		http.Error(w, "No alerts in payload", http.StatusBadRequest)
		return
	}

	// Validate each alert has required fields
	for i, alert := range alertPayload.Alerts {
		if alert.Labels == nil {
			log.Printf("Alert %d missing labels", i)
			alertPayload.Alerts = append(alertPayload.Alerts[:i], alertPayload.Alerts[i+1:]...)
			continue
		}
		if alert.Labels["alertname"] == "" {
			log.Printf("Alert %d missing alertname label", i)
			alertPayload.Alerts = append(alertPayload.Alerts[:i], alertPayload.Alerts[i+1:]...)
			continue
		}
	}

	// Re-check after filtering invalid alerts
	if len(alertPayload.Alerts) == 0 {
		http.Error(w, "No valid alerts in payload", http.StatusBadRequest)
		return
	}

	log.Printf("Received %d alerts from %s", len(alertPayload.Alerts), alertPayload.Receiver)

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
			log.Printf("Skipping alert %s: missing service_name label", alert.Labels["alertname"])
			continue
		}

		if alert.Status == "resolved" {
			log.Printf("Processing RESOLVED alert %s for service %s", alert.Labels["alertname"], serviceName)
			if h.generator == nil || h.orchestrator == nil {
				continue
			}

			// Prepare context mapping back to incident start for full postmortem view
			ctx, err := h.orchestrator.PrepareContext(context.Background(), serviceName, alert.StartsAt)
			if err != nil {
				log.Printf("Failed to prepare context for postmortem on %s: %v", serviceName, err)
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
				log.Printf("Failed to generate postmortem for %s: %v", serviceName, err)
				continue
			}

			log.Printf("Generated Postmortem ID: %s for service: %s", pm.ID, serviceName)

			// Resolve incident in database if available
			if h.database != nil {
				if err := h.database.ResolveIncident(pm.ID, pm.RootCause, pm.Markdown); err != nil {
					log.Printf("Failed to resolve incident in database: %v", err)
				} else {
					log.Printf("Resolved incident %s in database", pm.ID)
				}
			}

			if h.mdReporter != nil {
				if err := h.mdReporter.SendPostmortem(pm); err != nil {
					log.Printf("Failed to save postmortem markdown: %v", err)
				}
			}
			continue
		}

		if alert.Status != "firing" {
			continue
		}

		log.Printf("Processing alert %s for service %s", alert.Labels["alertname"], serviceName)

		// Guard against nil dependencies (for tests)
		if h.orchestrator == nil || h.analyzer == nil {
			log.Printf("Skipping alert processing: missing orchestrator or analyzer")
			continue
		}

		// Create analysis context with metrics, logs, commits, and traces
		ctx, err := h.orchestrator.PrepareContext(context.Background(), serviceName, alert.StartsAt)
		if err != nil {
			log.Printf("Failed to prepare context for %s: %v", serviceName, err)
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
			log.Printf("Failed to analyze alert for %s: %v", serviceName, err)
			continue
		}

		log.Printf("Analysis complete for %s: %s", serviceName, result.Summary)

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
				log.Printf("Failed to create incident in database: %v", err)
			} else {
				log.Printf("Created incident %s in database", result.ID)
			}
		}

		// Send to output channels (Slack and Markdown)
		if h.slackSender != nil {
			if err := h.slackSender.SendAnalysis(result); err != nil {
				log.Printf("Failed to send Slack notification: %v", err)
			} else {
				log.Printf("Sent Slack notification for %s", serviceName)
			}
		}

		if h.mdReporter != nil {
			if err := h.mdReporter.Report(result); err != nil {
				log.Printf("Failed to save analysis markdown: %v", err)
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
		log.Printf("Failed to list incidents: %v", err)
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
		log.Printf("Failed to get incident: %v", err)
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
