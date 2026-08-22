package theme

import (
	catppuccin "github.com/catppuccin/go"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// CatppuccinTheme implements the Theme interface with Catppuccin colors.
// It provides both dark (Mocha) and light (Latte) variants.
type CatppuccinTheme struct {
	BaseTheme
}

// NewCatppuccinTheme creates a new instance of the Catppuccin theme.
func NewCatppuccinTheme() *CatppuccinTheme {
	// Get the Catppuccin palettes
	mocha := catppuccin.Mocha
	latte := catppuccin.Latte

	theme := &CatppuccinTheme{}

	// Base colors
	theme.PrimaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Blue().Hex),
		Light: lipgloss.Color(latte.Blue().Hex),
	}
	theme.SecondaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Mauve().Hex),
		Light: lipgloss.Color(latte.Mauve().Hex),
	}
	theme.AccentColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Peach().Hex),
		Light: lipgloss.Color(latte.Peach().Hex),
	}

	// Status colors
	theme.ErrorColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Red().Hex),
		Light: lipgloss.Color(latte.Red().Hex),
	}
	theme.WarningColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Peach().Hex),
		Light: lipgloss.Color(latte.Peach().Hex),
	}
	theme.SuccessColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Green().Hex),
		Light: lipgloss.Color(latte.Green().Hex),
	}
	theme.InfoColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Blue().Hex),
		Light: lipgloss.Color(latte.Blue().Hex),
	}

	// Text colors
	theme.TextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Text().Hex),
		Light: lipgloss.Color(latte.Text().Hex),
	}
	theme.TextMutedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Subtext0().Hex),
		Light: lipgloss.Color(latte.Subtext0().Hex),
	}
	theme.TextEmphasizedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Lavender().Hex),
		Light: lipgloss.Color(latte.Lavender().Hex),
	}

	// Background colors
	theme.BackgroundColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#212121"), // From existing styles
		Light: lipgloss.Color("#EEEEEE"), // Light equivalent
	}
	theme.BackgroundSecondaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#2c2c2c"), // From existing styles
		Light: lipgloss.Color("#E0E0E0"), // Light equivalent
	}
	theme.BackgroundDarkerColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#181818"), // From existing styles
		Light: lipgloss.Color("#F5F5F5"), // Light equivalent
	}

	// Border colors
	theme.BorderNormalColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#4b4c5c"), // From existing styles
		Light: lipgloss.Color("#BDBDBD"), // Light equivalent
	}
	theme.BorderFocusedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Blue().Hex),
		Light: lipgloss.Color(latte.Blue().Hex),
	}
	theme.BorderDimColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Surface0().Hex),
		Light: lipgloss.Color(latte.Surface0().Hex),
	}

	// Diff view colors
	theme.DiffAddedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#478247"), // From existing diff.go
		Light: lipgloss.Color("#2E7D32"), // Light equivalent
	}
	theme.DiffRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#7C4444"), // From existing diff.go
		Light: lipgloss.Color("#C62828"), // Light equivalent
	}
	theme.DiffContextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#a0a0a0"), // From existing diff.go
		Light: lipgloss.Color("#757575"), // Light equivalent
	}
	theme.DiffHunkHeaderColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#a0a0a0"), // From existing diff.go
		Light: lipgloss.Color("#757575"), // Light equivalent
	}
	theme.DiffHighlightAddedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#DAFADA"), // From existing diff.go
		Light: lipgloss.Color("#A5D6A7"), // Light equivalent
	}
	theme.DiffHighlightRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#FADADD"), // From existing diff.go
		Light: lipgloss.Color("#EF9A9A"), // Light equivalent
	}
	theme.DiffAddedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#303A30"), // From existing diff.go
		Light: lipgloss.Color("#E8F5E9"), // Light equivalent
	}
	theme.DiffRemovedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#3A3030"), // From existing diff.go
		Light: lipgloss.Color("#FFEBEE"), // Light equivalent
	}
	theme.DiffContextBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#212121"), // From existing diff.go
		Light: lipgloss.Color("#F5F5F5"), // Light equivalent
	}
	theme.DiffLineNumberColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#888888"), // From existing diff.go
		Light: lipgloss.Color("#9E9E9E"), // Light equivalent
	}
	theme.DiffAddedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#293229"), // From existing diff.go
		Light: lipgloss.Color("#C8E6C9"), // Light equivalent
	}
	theme.DiffRemovedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#332929"), // From existing diff.go
		Light: lipgloss.Color("#FFCDD2"), // Light equivalent
	}

	// Markdown colors
	theme.MarkdownTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Text().Hex),
		Light: lipgloss.Color(latte.Text().Hex),
	}
	theme.MarkdownHeadingColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Mauve().Hex),
		Light: lipgloss.Color(latte.Mauve().Hex),
	}
	theme.MarkdownLinkColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Sky().Hex),
		Light: lipgloss.Color(latte.Sky().Hex),
	}
	theme.MarkdownLinkTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Pink().Hex),
		Light: lipgloss.Color(latte.Pink().Hex),
	}
	theme.MarkdownCodeColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Green().Hex),
		Light: lipgloss.Color(latte.Green().Hex),
	}
	theme.MarkdownBlockQuoteColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Yellow().Hex),
		Light: lipgloss.Color(latte.Yellow().Hex),
	}
	theme.MarkdownEmphColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Yellow().Hex),
		Light: lipgloss.Color(latte.Yellow().Hex),
	}
	theme.MarkdownStrongColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Peach().Hex),
		Light: lipgloss.Color(latte.Peach().Hex),
	}
	theme.MarkdownHorizontalRuleColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Overlay0().Hex),
		Light: lipgloss.Color(latte.Overlay0().Hex),
	}
	theme.MarkdownListItemColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Blue().Hex),
		Light: lipgloss.Color(latte.Blue().Hex),
	}
	theme.MarkdownListEnumerationColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Sky().Hex),
		Light: lipgloss.Color(latte.Sky().Hex),
	}
	theme.MarkdownImageColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Sapphire().Hex),
		Light: lipgloss.Color(latte.Sapphire().Hex),
	}
	theme.MarkdownImageTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Pink().Hex),
		Light: lipgloss.Color(latte.Pink().Hex),
	}
	theme.MarkdownCodeBlockColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Text().Hex),
		Light: lipgloss.Color(latte.Text().Hex),
	}

	// Syntax highlighting colors
	theme.SyntaxCommentColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Overlay1().Hex),
		Light: lipgloss.Color(latte.Overlay1().Hex),
	}
	theme.SyntaxKeywordColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Pink().Hex),
		Light: lipgloss.Color(latte.Pink().Hex),
	}
	theme.SyntaxFunctionColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Green().Hex),
		Light: lipgloss.Color(latte.Green().Hex),
	}
	theme.SyntaxVariableColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Sky().Hex),
		Light: lipgloss.Color(latte.Sky().Hex),
	}
	theme.SyntaxStringColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Yellow().Hex),
		Light: lipgloss.Color(latte.Yellow().Hex),
	}
	theme.SyntaxNumberColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Teal().Hex),
		Light: lipgloss.Color(latte.Teal().Hex),
	}
	theme.SyntaxTypeColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Sky().Hex),
		Light: lipgloss.Color(latte.Sky().Hex),
	}
	theme.SyntaxOperatorColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Pink().Hex),
		Light: lipgloss.Color(latte.Pink().Hex),
	}
	theme.SyntaxPunctuationColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(mocha.Text().Hex),
		Light: lipgloss.Color(latte.Text().Hex),
	}

	return theme
}

