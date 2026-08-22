package theme

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// TokyoNightTheme implements the Theme interface with Tokyo Night colors.
// It provides both dark and light variants.
type TokyoNightTheme struct {
	BaseTheme
}

// NewTokyoNightTheme creates a new instance of the Tokyo Night theme.
func NewTokyoNightTheme() *TokyoNightTheme {
	// Tokyo Night color palette
	// Dark mode colors
	darkBackground := "#222436"
	darkCurrentLine := "#1e2030"
	darkSelection := "#2f334d"
	darkForeground := "#c8d3f5"
	darkComment := "#636da6"
	darkRed := "#ff757f"
	darkOrange := "#ff966c"
	darkYellow := "#ffc777"
	darkGreen := "#c3e88d"
	darkCyan := "#86e1fc"
	darkBlue := "#82aaff"
	darkPurple := "#c099ff"
	darkBorder := "#3b4261"

	// Light mode colors (Tokyo Night Day)
	lightBackground := "#e1e2e7"
	lightCurrentLine := "#d5d6db"
	lightSelection := "#c8c9ce"
	lightForeground := "#3760bf"
	lightComment := "#848cb5"
	lightRed := "#f52a65"
	lightOrange := "#b15c00"
	lightYellow := "#8c6c3e"
	lightGreen := "#587539"
	lightCyan := "#007197"
	lightBlue := "#2e7de9"
	lightPurple := "#9854f1"
	lightBorder := "#a8aecb"

	theme := &TokyoNightTheme{}

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
		Dark:  lipgloss.Color("#191B29"), // Darker background from palette
		Light: lipgloss.Color("#f0f0f5"), // Slightly lighter than background
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
		Dark:  lipgloss.Color("#4fd6be"), // teal from palette
		Light: lipgloss.Color("#1e725c"),
	}
	theme.DiffRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#c53b53"), // red1 from palette
		Light: lipgloss.Color("#c53b53"),
	}
	theme.DiffContextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#828bb8"), // fg_dark from palette
		Light: lipgloss.Color("#7086b5"),
	}
	theme.DiffHunkHeaderColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#828bb8"), // fg_dark from palette
		Light: lipgloss.Color("#7086b5"),
	}
	theme.DiffHighlightAddedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#b8db87"), // git.add from palette
		Light: lipgloss.Color("#4db380"),
	}
	theme.DiffHighlightRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#e26a75"), // git.delete from palette
		Light: lipgloss.Color("#f52a65"),
	}
	theme.DiffAddedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#20303b"),
		Light: lipgloss.Color("#d5e5d5"),
	}
	theme.DiffRemovedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#37222c"),
		Light: lipgloss.Color("#f7d8db"),
	}
	theme.DiffContextBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(darkBackground),
		Light: lipgloss.Color(lightBackground),
	}
	theme.DiffLineNumberColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#545c7e"), // dark3 from palette
		Light: lipgloss.Color("#848cb5"),
	}
	theme.DiffAddedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#1b2b34"),
		Light: lipgloss.Color("#c5d5c5"),
	}
	theme.DiffRemovedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#2d1f26"),
		Light: lipgloss.Color("#e7c8cb"),
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

