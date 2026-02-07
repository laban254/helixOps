package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"helixops/internal/analyzer"
	"helixops/internal/config"
	"helixops/internal/models"
	"helixops/internal/orchestrator"

	"github.com/go-chi/chi/v5"
)

// Handler holds the server dependencies
type Handler struct {
	cfg         *config.Config
	orchestrator *orchestrator.Orchestrator
	analyzer    *analyzer.Analyzer
}

// NewHandler creates a new handler
func NewHandler(cfg *config.Config, orch *orchestrator.Orchestrator, anlz *analyzer.Analyzer) *Handler {
	return &Handler{
		cfg:         cfg,
		orchestrator: orch,
		analyzer:    anlz,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/webhook", h.HandleWebhook)
	r.Get("/health", h.HandleHealth)
	r.Get("/ready", h.HandleReady)
}

// HandleWebhook receives alerts from Prometheus AlertManager
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
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

	log.Printf("Received %d alerts from %s", len(alertPayload.Alerts), alertPayload.Receiver)

	// Process alerts asynchronously
	go h.processAlerts(alertPayload)

	// Acknowledge immediately
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "accepted",
		"message": fmt.Sprintf("Processing %d alerts", len(alertPayload.Alerts)),
	})
}

// processAlerts processes alerts asynchronously
func (h *Handler) processAlerts(payload models.AlertManagerPayload) {
	for _, alert := range payload.Alerts {
		if alert.Status != "firing" {
			continue
		}

		serviceName := extractServiceName(alert.Labels)
		if serviceName == "" {
			log.Printf("Skipping alert %s: missing service_name label", alert.Labels["alertname"])
			continue
		}

		log.Printf("Processing alert %s for service %s", alert.Labels["alertname"], serviceName)

		// Create analysis context
		ctx, err := h.orchestrator.PrepareContext(r.Context(), serviceName, alert.StartsAt)
		if err != nil {
			log.Printf("Failed to prepare context for %s: %v", serviceName, err)
			continue
		}

		// Analyze with LLM
		result, err := h.analyzer.Analyze(ctx, alert)
		if err != nil {
			log.Printf("Failed to analyze alert for %s: %v", serviceName, err)
			continue
		}

		log.Printf("Analysis complete for %s: %s", serviceName, result.Summary)

		// TODO: Send to output channels (Slack, Markdown)
		_ = result
	}
}

// extractServiceName extracts the service name from alert labels
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
	// TODO: Check if all clients are ready
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
	})
}
