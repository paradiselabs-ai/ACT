package core

import (
	"strings"
	"testing"
)

func TestFormatTokensAndCost(t *testing.T) {
	tests := []struct {
		name          string
		tokens        int64
		contextWindow int64
		cost          float64
		wantContains  []string
		wantNotContains []string
	}{
		{
			name:            "context window zero",
			tokens:          5500,
			contextWindow:   0,
			cost:            1.23,
			wantContains:    []string{"5.5K", "$1.23"},
			wantNotContains: []string{"█", "░"},
		},
		{
			name:            "context window positive",
			tokens:          5500,
			contextWindow:   10000,
			cost:            1.23,
			wantContains:    []string{"█", "░", "5.5K", "$1.23"},
			wantNotContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTokensAndCost(tt.tokens, tt.contextWindow, tt.cost)
			for _, s := range tt.wantContains {
				if !strings.Contains(got, s) {
					t.Errorf("formatTokensAndCost() = %q, expected to contain %q", got, s)
				}
			}
			for _, s := range tt.wantNotContains {
				if strings.Contains(got, s) {
					t.Errorf("formatTokensAndCost() = %q, expected NOT to contain %q", got, s)
				}
			}
		})
	}
}
