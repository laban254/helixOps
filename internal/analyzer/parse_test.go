package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLLMResponse(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		wantConf     string
		wantSteps    []string
		rootContains string
	}{
		{
			name:         "canonical format",
			response:     "# Incident\n**Confidence Score:** 80%\nbody\n## 4. Recommended Action\n- Do X\n- Do Y",
			wantConf:     "80%",
			wantSteps:    []string{"Do X", "Do Y"},
			rootContains: "body",
		},
		{
			name:      "alternate heading and numbered list",
			response:  "Confidence: high\nanalysis here\n### Next Steps\n1. Restart pod\n2. Roll back deploy\n## Appendix\n- ignored",
			wantConf:  "high",
			wantSteps: []string{"Restart pod", "Roll back deploy"},
		},
		{
			name:      "no heading hash, action items synonym",
			response:  "**Confidence** - medium\nstuff\nAction Items:\n* check disk\n* page oncall",
			wantConf:  "medium",
			wantSteps: []string{"check disk", "page oncall"},
		},
		{
			name:         "missing actions section falls back",
			response:     "Just a free-form root cause with no structure at all.",
			wantConf:     "medium",
			wantSteps:    nil,
			rootContains: "free-form root cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, conf, steps := parseLLMResponse(tt.response)
			assert.Equal(t, tt.wantConf, conf)
			assert.Equal(t, tt.wantSteps, steps)
			if tt.rootContains != "" {
				assert.Contains(t, root, tt.rootContains)
			}
		})
	}
}
