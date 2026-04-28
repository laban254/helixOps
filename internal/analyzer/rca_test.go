package analyzer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"helixops/internal/models"

	"github.com/stretchr/testify/require"
)

// mock provider that returns a fixed response
type mockProvider struct{}

func (m *mockProvider) Analyze(ctx context.Context, prompt string) (string, error) {
	// Return a minimal structured response
	return "**Confidence Score:** 80\n## 4. Recommended Action\n- Do X\n- Do Y", nil
}

func (m *mockProvider) Name() string { return "mock" }

func TestAnalyzeWithMissingData(t *testing.T) {
	a := New(&mockProvider{})

	ctx := &models.AnalysisContext{
		ServiceName: "test-service",
		Alert: models.AlertInfo{
			Name:      "HighLatency",
			Severity:  "critical",
			Summary:   "latency spike",
			StartedAt: time.Now(),
		},
		// Simulate Prometheus down
		Errors: map[string]string{"prometheus": "timeout"},
	}

	res, err := a.AnalyzeWithContext(context.Background(), ctx)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Ensure analysis used context and produced next steps
	b, _ := json.Marshal(res)
	require.Contains(t, string(b), "Do X")
}
