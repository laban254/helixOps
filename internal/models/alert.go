// Package models defines the shared core data structures used throughout the HelixOps agent.
package models

import "time"

// AlertManagerPayload represents the Prometheus AlertManager webhook payload
type AlertManagerPayload struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []AlertItem       `json:"alerts"`
}

// AlertItem represents a single alert from AlertManager
type AlertItem struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// IsFiring returns true if the alert is currently firing
func (a *AlertItem) IsFiring() bool {
	return a.Status == "firing"
}

// GetLabel returns the value of a label, empty string if not found
func (a *AlertItem) GetLabel(key string) string {
	if a.Labels == nil {
		return ""
	}
	return a.Labels[key]
}

// GetAnnotation returns the value of an annotation, empty string if not found
func (a *AlertItem) GetAnnotation(key string) string {
	if a.Annotations == nil {
		return ""
	}
	return a.Annotations[key]
}
