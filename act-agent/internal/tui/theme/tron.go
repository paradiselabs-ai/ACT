package theme

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// TronTheme implements the Theme interface with Tron-inspired colors.
// It provides both dark and light variants, though Tron is primarily a dark theme.
type TronTheme struct {
	BaseTheme
}

// NewTronTheme creates a new instance of the Tron theme.
func NewTronTheme() *TronTheme {
	// Tron color palette
	// Inspired by the Tron movie's neon aesthetic
	darkBackground := "#0c141f"
	darkCurrentLine := "#1a2633"
	darkSelection := "#1a2633"
	darkForeground := "#caf0ff"
	darkComment := "#4d6b87"
	darkCyan := "#00d9ff"
	darkBlue := "#007fff"
	darkOrange := "#ff9000"
	darkPink := "#ff00a0"
	darkPurple := "#b73fff"
	darkRed := "#ff3333"
	darkYellow := "#ffcc00"
	darkGreen := "#00ff8f"
	darkBorder := "#1a2633"

	// Light mode approximation
	lightBackground := "#f0f8ff"
	lightCurrentLine := "#e0f0ff"
	lightSelection := "#d0e8ff"
	lightForeground := "#0c141f"
	lightComment := "#4d6b87"
	lightCyan := "#0097b3"
	lightBlue := "#0066cc"
	lightOrange := "#cc7300"
	lightPink := "#cc0080"
	lightPurple := "#9932cc"
	lightRed := "#cc2929"
	lightYellow := "#cc9900"
	lightGreen := "#00cc72"
	lightBorder := "#d0e8ff"

	theme := &TronTheme{}

	// Base colors
	theme.PrimaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.SecondaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
	}
	theme.AccentColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkOrange),
		Light: lipgloss.Color(lightOrange),
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
		Dark:  lipgloss.Color("#070d14"), // Slightly darker than background
		Light: lipgloss.Color("#ffffff"), // Slightly lighter than background
	}

	// Border colors
	theme.BorderNormalColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBorder),
		Light: lipgloss.Color(lightBorder),
	}
	theme.BorderFocusedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.BorderDimColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkSelection),
		Light: lipgloss.Color(lightSelection),
	}

	// Diff view colors
	theme.DiffAddedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkGreen),
		Light: lipgloss.Color(lightGreen),
	}
	theme.DiffRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkRed),
		Light: lipgloss.Color(lightRed),
	}
	theme.DiffContextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkComment),
		Light: lipgloss.Color(lightComment),
	}
	theme.DiffHunkHeaderColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
	}
	theme.DiffHighlightAddedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#00ff8f"),
		Light: lipgloss.Color("#a5d6a7"),
	}
	theme.DiffHighlightRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#ff3333"),
		Light: lipgloss.Color("#ef9a9a"),
	}
	theme.DiffAddedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#0a2a1a"),
		Light: lipgloss.Color("#e8f5e9"),
	}
	theme.DiffRemovedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#2a0a0a"),
		Light: lipgloss.Color("#ffebee"),
	}
	theme.DiffContextBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBackground),
		Light: lipgloss.Color(lightBackground),
	}
	theme.DiffLineNumberColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkComment),
		Light: lipgloss.Color(lightComment),
	}
	theme.DiffAddedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#082015"),
		Light: lipgloss.Color("#c8e6c9"),
	}
	theme.DiffRemovedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#200808"),
		Light: lipgloss.Color("#ffcdd2"),
	}

	// Markdown colors
	theme.MarkdownTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkForeground),
		Light: lipgloss.Color(lightForeground),
	}
	theme.MarkdownHeadingColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.MarkdownLinkColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
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
		Dark:  lipgloss.Color(darkOrange),
		Light: lipgloss.Color(lightOrange),
	}
	theme.MarkdownHorizontalRuleColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkComment),
		Light: lipgloss.Color(lightComment),
	}
	theme.MarkdownListItemColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
	}
	theme.MarkdownListEnumerationColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.MarkdownImageColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
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
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.SyntaxFunctionColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkGreen),
		Light: lipgloss.Color(lightGreen),
	}
	theme.SyntaxVariableColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkOrange),
		Light: lipgloss.Color(lightOrange),
	}
	theme.SyntaxStringColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkYellow),
		Light: lipgloss.Color(lightYellow),
	}
	theme.SyntaxNumberColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
	}
	theme.SyntaxTypeColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkPurple),
		Light: lipgloss.Color(lightPurple),
	}
	theme.SyntaxOperatorColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkPink),
		Light: lipgloss.Color(lightPink),
	}
	theme.SyntaxPunctuationColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkForeground),
		Light: lipgloss.Color(lightForeground),
	}

	return theme
}

