package styles

import (
	"fmt"
	"image/color"
	"os"
	"sync"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
)

const defaultMargin = 1

// Helper functions for style pointers
func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint       { return &u }

// Cached Glamour TermRenderers keyed by width. Creating a new renderer is
// expensive (compiles the full ansi.StyleConfig + Chroma syntax rules) — on
// a busy session we were paying 100-300ms per GetMarkdownRenderer call, and
// every tool-call re-render hit this path. One renderer per width amortizes
// the construction cost across the session. Thread-safe via sync.Map.
var markdownRendererCache sync.Map // map[int]*glamour.TermRenderer

// GetMarkdownRenderer returns a Glamour renderer for the given terminal width.
// Renderers are cached per width. The renderer's internal state is safe for
// concurrent Render() calls; the cache itself is safe via sync.Map.
func GetMarkdownRenderer(width int) *glamour.TermRenderer {
	if cached, ok := markdownRendererCache.Load(width); ok {
		return cached.(*glamour.TermRenderer)
	}
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(generateMarkdownStyleConfig()),
		glamour.WithWordWrap(width),
	)
	actual, _ := markdownRendererCache.LoadOrStore(width, r)
	return actual.(*glamour.TermRenderer)
}

// InvalidateMarkdownRendererCache drops all cached renderers. Called on theme
// change so the next GetMarkdownRenderer rebuilds with the new style config.
func InvalidateMarkdownRendererCache() {
	resetDarkBackgroundCache()
	markdownRendererCache.Range(func(k, _ any) bool {
		markdownRendererCache.Delete(k)
		return true
	})
}

// creates an ansi.StyleConfig for markdown rendering
// using adaptive colors from the provided theme.
func generateMarkdownStyleConfig() ansi.StyleConfig {
	t := theme.CurrentTheme()

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "",
				BlockSuffix: "",
				Color:       stringPtr(adaptiveColorToString(t.MarkdownText())),
			},
			Margin: uintPtr(defaultMargin),
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  stringPtr(adaptiveColorToString(t.MarkdownBlockQuote())),
				Italic: boolPtr(true),
				Prefix: "┃ ",
			},
			Indent:      uintPtr(1),
			IndentToken: stringPtr(BaseStyle().Render(" ")),
		},
		List: ansi.StyleList{
			LevelIndent: defaultMargin,
			StyleBlock: ansi.StyleBlock{
				IndentToken: stringPtr(BaseStyle().Render(" ")),
				StylePrimitive: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.MarkdownText())),
				},
			},
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Color:       stringPtr(adaptiveColorToString(t.MarkdownHeading())),
				Bold:        boolPtr(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "█ ",
				Color:  stringPtr(adaptiveColorToString(t.MarkdownHeading())),
				Bold:   boolPtr(true),
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "■ ",
				Color:  stringPtr(adaptiveColorToString(t.MarkdownHeading())),
				Bold:   boolPtr(true),
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "▪ ",
				Color:  stringPtr(adaptiveColorToString(t.MarkdownHeading())),
				Bold:   boolPtr(true),
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "    ",
				Color:  stringPtr(adaptiveColorToString(t.MarkdownHeading())),
				Bold:   boolPtr(true),
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "      ",
				Color:  stringPtr(adaptiveColorToString(t.MarkdownHeading())),
				Bold:   boolPtr(true),
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "        ",
				Color:  stringPtr(adaptiveColorToString(t.MarkdownHeading())),
				Bold:   boolPtr(true),
			},
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut: boolPtr(true),
			Color:      stringPtr(adaptiveColorToString(t.TextMuted())),
		},
		Emph: ansi.StylePrimitive{
			Color:  stringPtr(adaptiveColorToString(t.MarkdownEmph())),
			Italic: boolPtr(true),
		},
		Strong: ansi.StylePrimitive{
			Bold:  boolPtr(true),
			Color: stringPtr(adaptiveColorToString(t.MarkdownStrong())),
		},
		HorizontalRule: ansi.StylePrimitive{
			Color:  stringPtr(adaptiveColorToString(t.MarkdownHorizontalRule())),
			Format: "\n─────────────────────────────────────────\n",
		},
		Item: ansi.StylePrimitive{
			BlockPrefix: "• ",
			Color:       stringPtr(adaptiveColorToString(t.MarkdownListItem())),
		},
		Enumeration: ansi.StylePrimitive{
			BlockPrefix: ". ",
			Color:       stringPtr(adaptiveColorToString(t.MarkdownListEnumeration())),
		},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{},
			Ticked:         "[✓] ",
			Unticked:       "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Color:     stringPtr(adaptiveColorToString(t.MarkdownLink())),
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: stringPtr(adaptiveColorToString(t.MarkdownLinkText())),
			Bold:  boolPtr(true),
		},
		Image: ansi.StylePrimitive{
			Color:     stringPtr(adaptiveColorToString(t.MarkdownImage())),
			Underline: boolPtr(true),
			Format:    "🖼 {{.text}}",
		},
		ImageText: ansi.StylePrimitive{
			Color:  stringPtr(adaptiveColorToString(t.MarkdownImageText())),
			Format: "{{.text}}",
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  stringPtr(adaptiveColorToString(t.MarkdownCode())),
				Prefix: "",
				Suffix: "",
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Prefix: " ",
					Color:  stringPtr(adaptiveColorToString(t.MarkdownCodeBlock())),
				},
				Margin: uintPtr(defaultMargin),
			},
			Chroma: &ansi.Chroma{
				Text: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.MarkdownText())),
				},
				Error: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.Error())),
				},
				Comment: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxComment())),
				},
				CommentPreproc: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxKeyword())),
				},
				Keyword: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxKeyword())),
				},
				KeywordReserved: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxKeyword())),
				},
				KeywordNamespace: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxKeyword())),
				},
				KeywordType: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxType())),
				},
				Operator: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxOperator())),
				},
				Punctuation: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxPunctuation())),
				},
				Name: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxVariable())),
				},
				NameBuiltin: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxVariable())),
				},
				NameTag: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxKeyword())),
				},
				NameAttribute: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxFunction())),
				},
				NameClass: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxType())),
				},
				NameConstant: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxVariable())),
				},
				NameDecorator: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxFunction())),
				},
				NameFunction: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxFunction())),
				},
				LiteralNumber: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxNumber())),
				},
				LiteralString: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxString())),
				},
				LiteralStringEscape: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.SyntaxKeyword())),
				},
				GenericDeleted: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.DiffRemoved())),
				},
				GenericEmph: ansi.StylePrimitive{
					Color:  stringPtr(adaptiveColorToString(t.MarkdownEmph())),
					Italic: boolPtr(true),
				},
				GenericInserted: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.DiffAdded())),
				},
				GenericStrong: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.MarkdownStrong())),
					Bold:  boolPtr(true),
				},
				GenericSubheading: ansi.StylePrimitive{
					Color: stringPtr(adaptiveColorToString(t.MarkdownHeading())),
				},
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					BlockPrefix: "\n",
					BlockSuffix: "\n",
				},
			},
			CenterSeparator: stringPtr("┼"),
			ColumnSeparator: stringPtr("│"),
			RowSeparator:    stringPtr("─"),
		},
		DefinitionDescription: ansi.StylePrimitive{
			BlockPrefix: "\n ❯ ",
			Color:       stringPtr(adaptiveColorToString(t.MarkdownLinkText())),
		},
		Text: ansi.StylePrimitive{
			Color: stringPtr(adaptiveColorToString(t.MarkdownText())),
		},
		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(adaptiveColorToString(t.MarkdownText())),
			},
		},
	}
}

// darkBackgroundCached caches the result of lipgloss.HasDarkBackground.
//
// lipgloss.HasDarkBackground sends a DCS query to the terminal and BLOCKS
// waiting for the response (OSC 11 on most terms, CSI 2026 on kitty-proto).
// On Ghostty the round-trip is ~200ms. generateMarkdownStyleConfig calls
// adaptiveColorToString ~50 times — 50 × 200ms = 10 seconds of blocked
// Update goroutine on first-render per width. That was the splash-screen
// freeze the user was hitting.
//
// Resolving once at init and reusing drops first-render to single-digit ms.
// The terminal's dark/light state doesn't change during a session; if a
// theme refresh is ever needed, call resetDarkBackgroundCache.
var (
	darkBackgroundCached bool
	darkBackgroundOnce   sync.Once
)

func isDarkBackground() bool {
	return IsDarkBackground()
}

// IsDarkBackground is the package-public wrapper around the cached query.
// Use this from any code that needs the terminal's dark/light state — it
// lazily resolves on first call and reuses thereafter, avoiding the DCS
// round-trip that HasDarkBackground blocks on.
func IsDarkBackground() bool {
	darkBackgroundOnce.Do(func() {
		darkBackgroundCached = lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	})
	return darkBackgroundCached
}

// resetDarkBackgroundCache clears the cached background-color query so the
// next adaptiveColorToString re-queries the terminal. Wire this into theme
// change if/when users can toggle terminal background dynamically.
func resetDarkBackgroundCache() {
	darkBackgroundOnce = sync.Once{}
}

// adaptiveColorToString converts a compat.AdaptiveColor to the appropriate
// hex color string based on the current terminal background
func adaptiveColorToString(c compat.AdaptiveColor) string {
	var col color.Color
	if isDarkBackground() {
		col = c.Dark
	} else {
		col = c.Light
	}
	// Convert color.Color to hex string
	if col == nil {
		return "#ffffff"
	}
	r, g, b, _ := col.RGBA()
	// RGBA returns values in range [0, 65535], convert to [0, 255]
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}
