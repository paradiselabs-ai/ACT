package navigator

import (
	"testing"
)

func TestGetBadge(t *testing.T) {
	tests := []struct {
		name     string
		modelStr string
		want     string
	}{
		{
			name:     "empty model",
			modelStr: "",
			want:     "H3",
		},
		{
			name:     "hermes model",
			modelStr: "hermes-3-llama-3.1-8b",
			want:     "H3",
		},
		{
			name:     "sonnet model",
			modelStr: "claude-3-5-sonnet-20241022",
			want:     "SN",
		},
		{
			name:     "gpt-4 model",
			modelStr: "gpt-4o",
			want:     "G4",
		},
		{
			name:     "claude-code backend",
			modelStr: "claude-code",
			want:     "CC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBadge(tt.modelStr)
			if got != tt.want {
				t.Errorf("getBadge(%q) = %q, want %q", tt.modelStr, got, tt.want)
			}
		})
	}
}
