package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlertItemIsFiring(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"firing alert", "firing", true},
		{"resolved alert", "resolved", false},
		{"pending alert", "pending", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := AlertItem{
				Status: tt.status,
			}
			assert.Equal(t, tt.expected, alert.IsFiring())
		})
	}
}

func TestAlertItemGetLabel(t *testing.T) {
	alert := AlertItem{
		Labels: map[string]string{
			"service_name": "test-service",
			"alertname":    "HighLatency",
		},
	}

	assert.Equal(t, "test-service", alert.GetLabel("service_name"))
	assert.Equal(t, "HighLatency", alert.GetLabel("alertname"))
	assert.Equal(t, "", alert.GetLabel("nonexistent"))
	assert.Equal(t, "", (&AlertItem{}).GetLabel("any"))
}

func TestAlertItemGetAnnotation(t *testing.T) {
	alert := AlertItem{
		Annotations: map[string]string{
			"summary": "High latency detected",
			"runbook": "https://example.com/runbook",
		},
	}

	assert.Equal(t, "High latency detected", alert.GetAnnotation("summary"))
	assert.Equal(t, "https://example.com/runbook", alert.GetAnnotation("runbook"))
	assert.Equal(t, "", alert.GetAnnotation("nonexistent"))
	assert.Equal(t, "", (&AlertItem{}).GetAnnotation("any"))
}
