package theme

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// ACTTheme implements the Theme interface with ACT brand colors.
// It provides both dark and light variants.
type ACTTheme struct {
	BaseTheme
}

// NewACTTheme creates a new instance of the ACT theme.
func NewACTTheme() *ACTTheme {
	// ACT color palette
	// Dark mode colors
	darkBackground := "#212121"
	darkCurrentLine := "#252525"
	darkSelection := "#303030"
	darkForeground := "#e0e0e0"
	darkComment := "#6a6a6a"
	darkPrimary := "#fab283"   // Primary orange/gold
	darkSecondary := "#5c9cf5" // Secondary blue
	darkAccent := "#9d7cd8"    // Accent purple
	darkRed := "#e06c75"       // Error red
	darkOrange := "#f5a742"    // Warning orange
	darkGreen := "#7fd88f"     // Success green
	darkCyan := "#56b6c2"      // Info cyan
	darkYellow := "#e5c07b"    // Emphasized text
	darkBorder := "#4b4c5c"    // Border color

	// Light mode colors
	lightBackground := "#f8f8f8"
	lightCurrentLine := "#f0f0f0"
	lightSelection := "#e5e5e6"
	lightForeground := "#2a2a2a"
	lightComment := "#8a8a8a"
	lightPrimary := "#3b7dd8"   // Primary blue
	lightSecondary := "#7b5bb6" // Secondary purple
	lightAccent := "#d68c27"    // Accent orange/gold
	lightRed := "#d1383d"       // Error red
	lightOrange := "#d68c27"    // Warning orange
	lightGreen := "#3d9a57"     // Success green
	lightCyan := "#318795"      // Info cyan
	lightYellow := "#b0851f"    // Emphasized text
	lightBorder := "#d3d3d3"    // Border color

	theme := &ACTTheme{}

	// Base colors
	theme.PrimaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkPrimary),
		Light: lipgloss.Color(lightPrimary),
	}
	theme.SecondaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkSecondary),
		Light: lipgloss.Color(lightSecondary),
	}
	theme.AccentColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkAccent),
		Light: lipgloss.Color(lightAccent),
	}

	// Status colors
	theme.ErrorColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkRed),
		Light: lipgloss.Color(lightRed),
	}
	theme.WarningColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkOrange),
		Light: lipgloss.Color(lightOrange),
	}
	theme.SuccessColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkGreen),
		Light: lipgloss.Color(lightGreen),
	}
	theme.InfoColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}

	// Text colors
	theme.TextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkForeground),
		Light: lipgloss.Color(lightForeground),
	}
	theme.TextMutedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkComment),
		Light: lipgloss.Color(lightComment),
	}
	theme.TextEmphasizedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkYellow),
		Light: lipgloss.Color(lightYellow),
	}

	// Background colors
	theme.BackgroundColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBackground),
		Light: lipgloss.Color(lightBackground),
	}
	theme.BackgroundSecondaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCurrentLine),
		Light: lipgloss.Color(lightCurrentLine),
	}
	theme.BackgroundDarkerColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#121212"), // Slightly darker than background
		Light: lipgloss.Color("#ffffff"), // Slightly lighter than background
	}

	// Border colors
	theme.BorderNormalColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBorder),
		Light: lipgloss.Color(lightBorder),
	}
	theme.BorderFocusedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkPrimary),
		Light: lipgloss.Color(lightPrimary),
	}
	theme.BorderDimColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkSelection),
		Light: lipgloss.Color(lightSelection),
	}

	// Diff view colors
	theme.DiffAddedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#478247"),
		Light: lipgloss.Color("#2E7D32"),
	}
	theme.DiffRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#7C4444"),
		Light: lipgloss.Color("#C62828"),
	}
	theme.DiffContextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#a0a0a0"),
		Light: lipgloss.Color("#757575"),
	}
	theme.DiffHunkHeaderColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#a0a0a0"),
		Light: lipgloss.Color("#757575"),
	}
	theme.DiffHighlightAddedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#DAFADA"),
		Light: lipgloss.Color("#A5D6A7"),
	}
	theme.DiffHighlightRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#FADADD"),
		Light: lipgloss.Color("#EF9A9A"),
	}
	theme.DiffAddedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#303A30"),
		Light: lipgloss.Color("#E8F5E9"),
	}
	theme.DiffRemovedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#3A3030"),
		Light: lipgloss.Color("#FFEBEE"),
	}
	theme.DiffContextBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBackground),
		Light: lipgloss.Color(lightBackground),
	}
	theme.DiffLineNumberColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#888888"),
		Light: lipgloss.Color("#9E9E9E"),
	}
	theme.DiffAddedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#293229"),
		Light: lipgloss.Color("#C8E6C9"),
	}
	theme.DiffRemovedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#332929"),
		Light: lipgloss.Color("#FFCDD2"),
	}

	// Markdown colors
	theme.MarkdownTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkForeground),
		Light: lipgloss.Color(lightForeground),
	}
	theme.MarkdownHeadingColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkSecondary),
		Light: lipgloss.Color(lightSecondary),
	}
	theme.MarkdownLinkColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkPrimary),
		Light: lipgloss.Color(lightPrimary),
	}
	theme.MarkdownLinkTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.MarkdownCodeColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkGreen),
		Light: lipgloss.Color(lightGreen),
	}
	theme.MarkdownBlockQuoteColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkYellow),
		Light: lipgloss.Color(lightYellow),
	}
	theme.MarkdownEmphColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkYellow),
		Light: lipgloss.Color(lightYellow),
	}
	theme.MarkdownStrongColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkAccent),
		Light: lipgloss.Color(lightAccent),
	}
	theme.MarkdownHorizontalRuleColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkComment),
		Light: lipgloss.Color(lightComment),
	}
	theme.MarkdownListItemColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkPrimary),
		Light: lipgloss.Color(lightPrimary),
	}
	theme.MarkdownListEnumerationColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.MarkdownImageColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkPrimary),
		Light: lipgloss.Color(lightPrimary),
	}
	theme.MarkdownImageTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.MarkdownCodeBlockColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkForeground),
		Light: lipgloss.Color(lightForeground),
	}

	// Syntax highlighting colors
	theme.SyntaxCommentColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkComment),
		Light: lipgloss.Color(lightComment),
	}
	theme.SyntaxKeywordColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkSecondary),
		Light: lipgloss.Color(lightSecondary),
	}
	theme.SyntaxFunctionColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkPrimary),
		Light: lipgloss.Color(lightPrimary),
	}
	theme.SyntaxVariableColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkRed),
		Light: lipgloss.Color(lightRed),
	}
	theme.SyntaxStringColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkGreen),
		Light: lipgloss.Color(lightGreen),
	}
	theme.SyntaxNumberColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkAccent),
		Light: lipgloss.Color(lightAccent),
	}
	theme.SyntaxTypeColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkYellow),
		Light: lipgloss.Color(lightYellow),
	}
	theme.SyntaxOperatorColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.SyntaxPunctuationColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkForeground),
		Light: lipgloss.Color(lightForeground),
	}

	return theme
}

func init() {
	// Register the ACT theme with the theme manager
	RegisterTheme("act", NewACTTheme())
	// Backward compatibility alias
	RegisterTheme("opencode", NewACTTheme())
}

