package styles

import "strings"

// ModelBadge returns the short 1-3 char badge for a model identifier.
// Single source of truth: the status strip (core/status.go) and the context
// navigator previously shipped two diverging implementations that rendered
// different badges for the same model (audit H12). Both now delegate here.
//
// The table is prefix-matched, lowercase, most-specific first. Unknown
// models fall back to a deterministic abbreviation of the name itself
// (first letters of the first two dash-separated segments, uppercased).
func ModelBadge(modelStr string) string {
	if modelStr == "" {
		return "-"
	}
	lower := strings.ToLower(modelStr)
	switch {
	case strings.Contains(lower, "hermes"):
		return "H3"
	case strings.Contains(lower, "claude-3-7-sonnet"):
		return "S7"
	case strings.Contains(lower, "sonnet"):
		return "SN"
	case strings.Contains(lower, "opus"):
		return "OP"
	case strings.Contains(lower, "haiku"):
		return "HK"
	case strings.Contains(lower, "gpt-4") || strings.Contains(lower, "gpt4"):
		return "G4"
	case strings.Contains(lower, "claude-code"), strings.Contains(lower, "claude"):
		return "CC"
	case strings.Contains(lower, "gemini-2.0-flash"):
		return "G2"
	case strings.Contains(lower, "gemini-1.5-pro"):
		return "G1P"
	case strings.Contains(lower, "gemini"):
		return "GM"
	case strings.Contains(lower, "deepseek-reasoner"), strings.Contains(lower, "deepseek-r1"):
		return "R1"
	case strings.Contains(lower, "deepseek-chat"), strings.Contains(lower, "deepseek-v3"):
		return "V3"
	case strings.Contains(lower, "llama-3.3-70b"):
		return "L70"
	case strings.Contains(lower, "llama"):
		return "L3"
	case strings.Contains(lower, "qwen2.5-coder"), strings.Contains(lower, "qwen"):
		return "QW"
	case strings.Contains(lower, "glm"):
		return "GLM"
	default:
		parts := strings.Split(modelStr, "-")
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return strings.ToUpper(parts[0][:1] + parts[1][:1])
		}
		if len(modelStr) >= 2 {
			return strings.ToUpper(modelStr[:2])
		}
		return strings.ToUpper(modelStr)
	}
}
