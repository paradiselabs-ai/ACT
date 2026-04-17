package theme

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// Gruvbox color palette constants
const (
	// Dark theme colors
	gruvboxDarkBg0          = "#282828"
	gruvboxDarkBg0Soft      = "#32302f"
	gruvboxDarkBg1          = "#3c3836"
	gruvboxDarkBg2          = "#504945"
	gruvboxDarkBg3          = "#665c54"
	gruvboxDarkBg4          = "#7c6f64"
	gruvboxDarkFg0          = "#fbf1c7"
	gruvboxDarkFg1          = "#ebdbb2"
	gruvboxDarkFg2          = "#d5c4a1"
	gruvboxDarkFg3          = "#bdae93"
	gruvboxDarkFg4          = "#a89984"
	gruvboxDarkGray         = "#928374"
	gruvboxDarkRed          = "#cc241d"
	gruvboxDarkRedBright    = "#fb4934"
	gruvboxDarkGreen        = "#98971a"
	gruvboxDarkGreenBright  = "#b8bb26"
	gruvboxDarkYellow       = "#d79921"
	gruvboxDarkYellowBright = "#fabd2f"
	gruvboxDarkBlue         = "#458588"
	gruvboxDarkBlueBright   = "#83a598"
	gruvboxDarkPurple       = "#b16286"
	gruvboxDarkPurpleBright = "#d3869b"
	gruvboxDarkAqua         = "#689d6a"
	gruvboxDarkAquaBright   = "#8ec07c"
	gruvboxDarkOrange       = "#d65d0e"
	gruvboxDarkOrangeBright = "#fe8019"

	// Light theme colors
	gruvboxLightBg0          = "#fbf1c7"
	gruvboxLightBg0Soft      = "#f2e5bc"
	gruvboxLightBg1          = "#ebdbb2"
	gruvboxLightBg2          = "#d5c4a1"
	gruvboxLightBg3          = "#bdae93"
	gruvboxLightBg4          = "#a89984"
	gruvboxLightFg0          = "#282828"
	gruvboxLightFg1          = "#3c3836"
	gruvboxLightFg2          = "#504945"
	gruvboxLightFg3          = "#665c54"
	gruvboxLightFg4          = "#7c6f64"
	gruvboxLightGray         = "#928374"
	gruvboxLightRed          = "#9d0006"
	gruvboxLightRedBright    = "#cc241d"
	gruvboxLightGreen        = "#79740e"
	gruvboxLightGreenBright  = "#98971a"
	gruvboxLightYellow       = "#b57614"
	gruvboxLightYellowBright = "#d79921"
	gruvboxLightBlue         = "#076678"
	gruvboxLightBlueBright   = "#458588"
	gruvboxLightPurple       = "#8f3f71"
	gruvboxLightPurpleBright = "#b16286"
	gruvboxLightAqua         = "#427b58"
	gruvboxLightAquaBright   = "#689d6a"
	gruvboxLightOrange       = "#af3a03"
	gruvboxLightOrangeBright = "#d65d0e"
)

// GruvboxTheme implements the Theme interface with Gruvbox colors.
// It provides both dark and light variants.
type GruvboxTheme struct {
	BaseTheme
}

// NewGruvboxTheme creates a new instance of the Gruvbox theme.
func NewGruvboxTheme() *GruvboxTheme {
	theme := &GruvboxTheme{}

	// Base colors
	theme.PrimaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBlueBright),
		Light: lipgloss.Color(gruvboxLightBlueBright),
	}
	theme.SecondaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkPurpleBright),
		Light: lipgloss.Color(gruvboxLightPurpleBright),
	}
	theme.AccentColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkOrangeBright),
		Light: lipgloss.Color(gruvboxLightOrangeBright),
	}

	// Status colors
	theme.ErrorColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkRedBright),
		Light: lipgloss.Color(gruvboxLightRedBright),
	}
	theme.WarningColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkYellowBright),
		Light: lipgloss.Color(gruvboxLightYellowBright),
	}
	theme.SuccessColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkGreenBright),
		Light: lipgloss.Color(gruvboxLightGreenBright),
	}
	theme.InfoColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBlueBright),
		Light: lipgloss.Color(gruvboxLightBlueBright),
	}

	// Text colors
	theme.TextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkFg1),
		Light: lipgloss.Color(gruvboxLightFg1),
	}
	theme.TextMutedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkFg4),
		Light: lipgloss.Color(gruvboxLightFg4),
	}
	theme.TextEmphasizedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkYellowBright),
		Light: lipgloss.Color(gruvboxLightYellowBright),
	}

	// Background colors
	theme.BackgroundColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBg0),
		Light: lipgloss.Color(gruvboxLightBg0),
	}
	theme.BackgroundSecondaryColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBg1),
		Light: lipgloss.Color(gruvboxLightBg1),
	}
	theme.BackgroundDarkerColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBg0Soft),
		Light: lipgloss.Color(gruvboxLightBg0Soft),
	}

	// Border colors
	theme.BorderNormalColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBg2),
		Light: lipgloss.Color(gruvboxLightBg2),
	}
	theme.BorderFocusedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBlueBright),
		Light: lipgloss.Color(gruvboxLightBlueBright),
	}
	theme.BorderDimColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBg1),
		Light: lipgloss.Color(gruvboxLightBg1),
	}

	// Diff view colors
	theme.DiffAddedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkGreenBright),
		Light: lipgloss.Color(gruvboxLightGreenBright),
	}
	theme.DiffRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkRedBright),
		Light: lipgloss.Color(gruvboxLightRedBright),
	}
	theme.DiffContextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkFg4),
		Light: lipgloss.Color(gruvboxLightFg4),
	}
	theme.DiffHunkHeaderColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkFg3),
		Light: lipgloss.Color(gruvboxLightFg3),
	}
	theme.DiffHighlightAddedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkGreenBright),
		Light: lipgloss.Color(gruvboxLightGreenBright),
	}
	theme.DiffHighlightRemovedColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkRedBright),
		Light: lipgloss.Color(gruvboxLightRedBright),
	}
	theme.DiffAddedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#3C4C3C"),  // Darker green background
		Light: lipgloss.Color("#E8F5E9"), // Light green background
	}
	theme.DiffRemovedBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#4C3C3C"),  // Darker red background
		Light: lipgloss.Color("#FFEBEE"), // Light red background
	}
	theme.DiffContextBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBg0),
		Light: lipgloss.Color(gruvboxLightBg0),
	}
	theme.DiffLineNumberColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkFg4),
		Light: lipgloss.Color(gruvboxLightFg4),
	}
	theme.DiffAddedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#32432F"),   // Slightly darker green
		Light: lipgloss.Color("#C8E6C9"), // Light green
	}
	theme.DiffRemovedLineNumberBgColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color("#43322F"),   // Slightly darker red
		Light: lipgloss.Color("#FFCDD2"), // Light red
	}

	// Markdown colors
	theme.MarkdownTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkFg1),
		Light: lipgloss.Color(gruvboxLightFg1),
	}
	theme.MarkdownHeadingColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkYellowBright),
		Light: lipgloss.Color(gruvboxLightYellowBright),
	}
	theme.MarkdownLinkColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBlueBright),
		Light: lipgloss.Color(gruvboxLightBlueBright),
	}
	theme.MarkdownLinkTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkAquaBright),
		Light: lipgloss.Color(gruvboxLightAquaBright),
	}
	theme.MarkdownCodeColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkGreenBright),
		Light: lipgloss.Color(gruvboxLightGreenBright),
	}
	theme.MarkdownBlockQuoteColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkAquaBright),
		Light: lipgloss.Color(gruvboxLightAquaBright),
	}
	theme.MarkdownEmphColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkYellowBright),
		Light: lipgloss.Color(gruvboxLightYellowBright),
	}
	theme.MarkdownStrongColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkOrangeBright),
		Light: lipgloss.Color(gruvboxLightOrangeBright),
	}
	theme.MarkdownHorizontalRuleColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBg3),
		Light: lipgloss.Color(gruvboxLightBg3),
	}
	theme.MarkdownListItemColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBlueBright),
		Light: lipgloss.Color(gruvboxLightBlueBright),
	}
	theme.MarkdownListEnumerationColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBlueBright),
		Light: lipgloss.Color(gruvboxLightBlueBright),
	}
	theme.MarkdownImageColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkPurpleBright),
		Light: lipgloss.Color(gruvboxLightPurpleBright),
	}
	theme.MarkdownImageTextColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkAquaBright),
		Light: lipgloss.Color(gruvboxLightAquaBright),
	}
	theme.MarkdownCodeBlockColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkFg1),
		Light: lipgloss.Color(gruvboxLightFg1),
	}

	// Syntax highlighting colors
	theme.SyntaxCommentColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkGray),
		Light: lipgloss.Color(gruvboxLightGray),
	}
	theme.SyntaxKeywordColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkRedBright),
		Light: lipgloss.Color(gruvboxLightRedBright),
	}
	theme.SyntaxFunctionColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkGreenBright),
		Light: lipgloss.Color(gruvboxLightGreenBright),
	}
	theme.SyntaxVariableColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkBlueBright),
		Light: lipgloss.Color(gruvboxLightBlueBright),
	}
	theme.SyntaxStringColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkYellowBright),
		Light: lipgloss.Color(gruvboxLightYellowBright),
	}
	theme.SyntaxNumberColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkPurpleBright),
		Light: lipgloss.Color(gruvboxLightPurpleBright),
	}
	theme.SyntaxTypeColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkYellow),
		Light: lipgloss.Color(gruvboxLightYellow),
	}
	theme.SyntaxOperatorColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkAquaBright),
		Light: lipgloss.Color(gruvboxLightAquaBright),
	}
	theme.SyntaxPunctuationColor = compat.AdaptiveColor{
		Dark:  lipgloss.Color(gruvboxDarkFg1),
		Light: lipgloss.Color(gruvboxLightFg1),
	}

	return theme
}

func init() {
	// Register the Gruvbox theme with the theme manager
	RegisterTheme("gruvbox", NewGruvboxTheme())
}