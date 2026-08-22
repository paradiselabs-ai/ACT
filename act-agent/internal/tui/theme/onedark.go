package theme

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// OneDarkTheme implements the Theme interface with Atom's One Dark colors.
// It provides both dark and light variants.
type OneDarkTheme struct {
	BaseTheme
}

// NewOneDarkTheme creates a new instance of the One Dark theme.
func NewOneDarkTheme() *OneDarkTheme {
	// One Dark color palette
	// Dark mode colors from Atom One Dark
	darkBackground := "#282c34"
	darkCurrentLine := "#2c313c"
	darkSelection := "#3e4451"
	darkForeground := "#abb2bf"
	darkComment := "#5c6370"
	darkRed := "#e06c75"
	darkOrange := "#d19a66"
	darkYellow := "#e5c07b"
	darkGreen := "#98c379"
	darkCyan := "#56b6c2"
	darkBlue := "#61afef"
	darkPurple := "#c678dd"
	darkBorder := "#3b4048"

	// Light mode colors from Atom One Light
	lightBackground := "#fafafa"
	lightCurrentLine := "#f0f0f0"
	lightSelection := "#e5e5e6"
	lightForeground := "#383a42"
	lightComment := "#a0a1a7"
	lightRed := "#e45649"
	lightOrange := "#da8548"
	lightYellow := "#c18401"
	lightGreen := "#50a14f"
	lightCyan := "#0184bc"
	lightBlue := "#4078f2"
	lightPurple := "#a626a4"
	lightBorder := "#d3d3d3"

	theme := &OneDarkTheme{}

	// Base colors
	theme.PrimaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
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
		Dark:  lipgloss.Color("#21252b"), // Slightly darker than background
		Light: lipgloss.Color("#ffffff"), // Slightly lighter than background
	}

	// Border colors
	theme.BorderNormalColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBorder),
		Light: lipgloss.Color(lightBorder),
	}
	theme.BorderFocusedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
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
		Dark:  lipgloss.Color(darkPurple),
		Light: lipgloss.Color(lightPurple),
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
		Dark:  lipgloss.Color(darkPurple),
		Light: lipgloss.Color(lightPurple),
	}
	theme.SyntaxFunctionColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBlue),
		Light: lipgloss.Color(lightBlue),
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
		Dark:  lipgloss.Color(darkOrange),
		Light: lipgloss.Color(lightOrange),
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

