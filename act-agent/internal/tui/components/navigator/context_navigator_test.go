package navigator

import (
	"testing"
)

func TestTruncateModelName(t *testing.T) {
	tests := []struct {
		name     string
		modelStr string
		width    int
		want     string
	}{
		{
			name:     "no model",
			modelStr: "",
			width:    30,
			want:     "",
		},
		{
			name:     "fits without truncation",
			modelStr: "gpt-4",
			width:    30,
			want:     "gpt-4",
		},
		{
			name:     "truncation needed",
			modelStr: "claude-3-5-sonnet-20241022",
			width:    30,
			want:     "claude-3-5..",
		},
		{
			name:     "extremely narrow width",
			modelStr: "gpt-4",
			width:    15,
			want:     "",
		},
		{
			name:     "consistent across roles — same width same result",
			modelStr: "hermes-3-llama-3.1-405b",
			width:    30,
			want:     "hermes-3-l..",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateModelName(tt.modelStr, tt.width)
			if got != tt.want {
				t.Errorf("truncateModelName(%q, %d) = %q, want %q", tt.modelStr, tt.width, got, tt.want)
			}
		})
	}
}
