package theme

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// MonokaiProTheme implements the Theme interface with Monokai Pro colors.
// It provides both dark and light variants.
type MonokaiProTheme struct {
	BaseTheme
}

// NewMonokaiProTheme creates a new instance of the Monokai Pro theme.
func NewMonokaiProTheme() *MonokaiProTheme {
	// Monokai Pro color palette (dark mode)
	darkBackground := "#2d2a2e"
	darkCurrentLine := "#403e41"
	darkSelection := "#5b595c"
	darkForeground := "#fcfcfa"
	darkComment := "#727072"
	darkRed := "#ff6188"
	darkOrange := "#fc9867"
	darkYellow := "#ffd866"
	darkGreen := "#a9dc76"
	darkCyan := "#78dce8"
	darkBlue := "#ab9df2"
	darkPurple := "#ab9df2"
	darkBorder := "#403e41"

	// Light mode colors (adapted from dark)
	lightBackground := "#fafafa"
	lightCurrentLine := "#f0f0f0"
	lightSelection := "#e5e5e6"
	lightForeground := "#2d2a2e"
	lightComment := "#939293"
	lightRed := "#f92672"
	lightOrange := "#fd971f"
	lightYellow := "#e6db74"
	lightGreen := "#9bca65"
	lightCyan := "#66d9ef"
	lightBlue := "#7e75db"
	lightPurple := "#ae81ff"
	lightBorder := "#d3d3d3"

	theme := &MonokaiProTheme{}

	// Base colors
	theme.PrimaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.SecondaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkPurple),
		Light: lipgloss.Color(lightPurple),
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
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
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
		Dark:  lipgloss.Color("#221f22"), // Slightly darker than background
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
		Dark:  lipgloss.Color("#a9dc76"),
		Light: lipgloss.Color("#9bca65"),
	}
	theme.DiffRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#ff6188"),
		Light: lipgloss.Color("#f92672"),
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
		Dark:  lipgloss.Color("#c2e7a9"),
		Light: lipgloss.Color("#c5e0b4"),
	}
	theme.DiffHighlightRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#ff8ca6"),
		Light: lipgloss.Color("#ffb3c8"),
	}
	theme.DiffAddedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#3a4a35"),
		Light: lipgloss.Color("#e8f5e9"),
	}
	theme.DiffRemovedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#4a3439"),
		Light: lipgloss.Color("#ffebee"),
	}
	theme.DiffContextBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBackground),
		Light: lipgloss.Color(lightBackground),
	}
	theme.DiffLineNumberColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#888888"),
		Light: lipgloss.Color("#9e9e9e"),
	}
	theme.DiffAddedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#2d3a28"),
		Light: lipgloss.Color("#c8e6c9"),
	}
	theme.DiffRemovedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#3d2a2e"),
		Light: lipgloss.Color("#ffcdd2"),
	}

	// Markdown colors
	theme.MarkdownTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkForeground),
		Light: lipgloss.Color(lightForeground),
	}
	theme.MarkdownHeadingColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkPurple),
		Light: lipgloss.Color(lightPurple),
	}
	theme.MarkdownLinkColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.MarkdownLinkTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
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
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.MarkdownListEnumerationColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
	}
	theme.MarkdownImageColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkCyan),
		Light: lipgloss.Color(lightCyan),
	}
	theme.MarkdownImageTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
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
		Dark:  lipgloss.Color(darkRed),
		Light: lipgloss.Color(lightRed),
	}
	theme.SyntaxFunctionColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkGreen),
		Light: lipgloss.Color(lightGreen),
	}
	theme.SyntaxVariableColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkForeground),
		Light: lipgloss.Color(lightForeground),
	}
	theme.SyntaxStringColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkYellow),
		Light: lipgloss.Color(lightYellow),
	}
	theme.SyntaxNumberColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkPurple),
		Light: lipgloss.Color(lightPurple),
	}
	theme.SyntaxTypeColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
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

