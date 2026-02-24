package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"helixops/internal/config"
	"helixops/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleWebhook(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
	}

	handler := NewHandler(cfg, nil, nil, nil, nil)
	router := SetupRouter(handler)

	// Create test alert payload
	payload := models.AlertManagerPayload{
		Version:  "4",
		Status:   "firing",
		Receiver: "helixops",
		Alerts: []models.AlertItem{
			{
				Status:      "firing",
				Labels:      map[string]string{"service_name": "test-service", "alertname": "HighLatency", "severity": "warning"},
				Annotations: map[string]string{"summary": "High latency detected"},
				StartsAt:    time.Now(),
			},
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "accepted", response["status"])
}

func TestHandleWebhookEmptyAlerts(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
	}

	handler := NewHandler(cfg, nil, nil, nil, nil)
	router := SetupRouter(handler)

	payload := models.AlertManagerPayload{
		Version: "4",
		Status:  "firing",
		Alerts:  []models.AlertItem{},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleWebhookInvalidPayload(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
	}

	handler := NewHandler(cfg, nil, nil, nil, nil)
	router := SetupRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleWebhookMethodNotAllowed(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
	}

	handler := NewHandler(cfg, nil, nil, nil, nil)
	router := SetupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleHealth(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
	}

	handler := NewHandler(cfg, nil, nil, nil, nil)
	router := SetupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.Contains(t, response, "timestamp")
}

func TestHandleReady(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
	}

	handler := NewHandler(cfg, nil, nil, nil, nil)
	router := SetupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ready", response["status"])
}
