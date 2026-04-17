package theme

import (
	"charm.land/lipgloss/v2/compat"
)

// Theme defines the interface for all UI themes in the application.
// All colors must be defined as compat.AdaptiveColor to support
// both light and dark terminal backgrounds.
type Theme interface {
	// Base colors
	Primary() compat.AdaptiveColor
	Secondary() compat.AdaptiveColor
	Accent() compat.AdaptiveColor

	// Status colors
	Error() compat.AdaptiveColor
	Warning() compat.AdaptiveColor
	Success() compat.AdaptiveColor
	Info() compat.AdaptiveColor

	// Text colors
	Text() compat.AdaptiveColor
	TextMuted() compat.AdaptiveColor
	TextEmphasized() compat.AdaptiveColor

	// Background colors
	Background() compat.AdaptiveColor
	BackgroundSecondary() compat.AdaptiveColor
	BackgroundDarker() compat.AdaptiveColor

	// Border colors
	BorderNormal() compat.AdaptiveColor
	BorderFocused() compat.AdaptiveColor
	BorderDim() compat.AdaptiveColor

	// Diff view colors
	DiffAdded() compat.AdaptiveColor
	DiffRemoved() compat.AdaptiveColor
	DiffContext() compat.AdaptiveColor
	DiffHunkHeader() compat.AdaptiveColor
	DiffHighlightAdded() compat.AdaptiveColor
	DiffHighlightRemoved() compat.AdaptiveColor
	DiffAddedBg() compat.AdaptiveColor
	DiffRemovedBg() compat.AdaptiveColor
	DiffContextBg() compat.AdaptiveColor
	DiffLineNumber() compat.AdaptiveColor
	DiffAddedLineNumberBg() compat.AdaptiveColor
	DiffRemovedLineNumberBg() compat.AdaptiveColor

	// Markdown colors
	MarkdownText() compat.AdaptiveColor
	MarkdownHeading() compat.AdaptiveColor
	MarkdownLink() compat.AdaptiveColor
	MarkdownLinkText() compat.AdaptiveColor
	MarkdownCode() compat.AdaptiveColor
	MarkdownBlockQuote() compat.AdaptiveColor
	MarkdownEmph() compat.AdaptiveColor
	MarkdownStrong() compat.AdaptiveColor
	MarkdownHorizontalRule() compat.AdaptiveColor
	MarkdownListItem() compat.AdaptiveColor
	MarkdownListEnumeration() compat.AdaptiveColor
	MarkdownImage() compat.AdaptiveColor
	MarkdownImageText() compat.AdaptiveColor
	MarkdownCodeBlock() compat.AdaptiveColor

	// Syntax highlighting colors
	SyntaxComment() compat.AdaptiveColor
	SyntaxKeyword() compat.AdaptiveColor
	SyntaxFunction() compat.AdaptiveColor
	SyntaxVariable() compat.AdaptiveColor
	SyntaxString() compat.AdaptiveColor
	SyntaxNumber() compat.AdaptiveColor
	SyntaxType() compat.AdaptiveColor
	SyntaxOperator() compat.AdaptiveColor
	SyntaxPunctuation() compat.AdaptiveColor
}

// BaseTheme provides a default implementation of the Theme interface
// that can be embedded in concrete theme implementations.
type BaseTheme struct {
	// Base colors
	PrimaryColor   compat.AdaptiveColor
	SecondaryColor compat.AdaptiveColor
	AccentColor    compat.AdaptiveColor

	// Status colors
	ErrorColor   compat.AdaptiveColor
	WarningColor compat.AdaptiveColor
	SuccessColor compat.AdaptiveColor
	InfoColor    compat.AdaptiveColor

	// Text colors
	TextColor           compat.AdaptiveColor
	TextMutedColor      compat.AdaptiveColor
	TextEmphasizedColor compat.AdaptiveColor

	// Background colors
	BackgroundColor          compat.AdaptiveColor
	BackgroundSecondaryColor compat.AdaptiveColor
	BackgroundDarkerColor    compat.AdaptiveColor

	// Border colors
	BorderNormalColor  compat.AdaptiveColor
	BorderFocusedColor compat.AdaptiveColor
	BorderDimColor     compat.AdaptiveColor

	// Diff view colors
	DiffAddedColor               compat.AdaptiveColor
	DiffRemovedColor             compat.AdaptiveColor
	DiffContextColor             compat.AdaptiveColor
	DiffHunkHeaderColor          compat.AdaptiveColor
	DiffHighlightAddedColor      compat.AdaptiveColor
	DiffHighlightRemovedColor    compat.AdaptiveColor
	DiffAddedBgColor             compat.AdaptiveColor
	DiffRemovedBgColor           compat.AdaptiveColor
	DiffContextBgColor           compat.AdaptiveColor
	DiffLineNumberColor          compat.AdaptiveColor
	DiffAddedLineNumberBgColor   compat.AdaptiveColor
	DiffRemovedLineNumberBgColor compat.AdaptiveColor

	// Markdown colors
	MarkdownTextColor            compat.AdaptiveColor
	MarkdownHeadingColor         compat.AdaptiveColor
	MarkdownLinkColor            compat.AdaptiveColor
	MarkdownLinkTextColor        compat.AdaptiveColor
	MarkdownCodeColor            compat.AdaptiveColor
	MarkdownBlockQuoteColor      compat.AdaptiveColor
	MarkdownEmphColor            compat.AdaptiveColor
	MarkdownStrongColor          compat.AdaptiveColor
	MarkdownHorizontalRuleColor  compat.AdaptiveColor
	MarkdownListItemColor        compat.AdaptiveColor
	MarkdownListEnumerationColor compat.AdaptiveColor
	MarkdownImageColor           compat.AdaptiveColor
	MarkdownImageTextColor       compat.AdaptiveColor
	MarkdownCodeBlockColor       compat.AdaptiveColor

	// Syntax highlighting colors
	SyntaxCommentColor     compat.AdaptiveColor
	SyntaxKeywordColor     compat.AdaptiveColor
	SyntaxFunctionColor    compat.AdaptiveColor
	SyntaxVariableColor    compat.AdaptiveColor
	SyntaxStringColor      compat.AdaptiveColor
	SyntaxNumberColor      compat.AdaptiveColor
	SyntaxTypeColor        compat.AdaptiveColor
	SyntaxOperatorColor    compat.AdaptiveColor
	SyntaxPunctuationColor compat.AdaptiveColor
}

// Implement the Theme interface for BaseTheme
func (t *BaseTheme) Primary() compat.AdaptiveColor   { return t.PrimaryColor }
func (t *BaseTheme) Secondary() compat.AdaptiveColor { return t.SecondaryColor }
func (t *BaseTheme) Accent() compat.AdaptiveColor    { return t.AccentColor }

func (t *BaseTheme) Error() compat.AdaptiveColor   { return t.ErrorColor }
func (t *BaseTheme) Warning() compat.AdaptiveColor { return t.WarningColor }
func (t *BaseTheme) Success() compat.AdaptiveColor { return t.SuccessColor }
func (t *BaseTheme) Info() compat.AdaptiveColor    { return t.InfoColor }

func (t *BaseTheme) Text() compat.AdaptiveColor           { return t.TextColor }
func (t *BaseTheme) TextMuted() compat.AdaptiveColor      { return t.TextMutedColor }
func (t *BaseTheme) TextEmphasized() compat.AdaptiveColor { return t.TextEmphasizedColor }

func (t *BaseTheme) Background() compat.AdaptiveColor          { return t.BackgroundColor }
func (t *BaseTheme) BackgroundSecondary() compat.AdaptiveColor { return t.BackgroundSecondaryColor }
func (t *BaseTheme) BackgroundDarker() compat.AdaptiveColor    { return t.BackgroundDarkerColor }

func (t *BaseTheme) BorderNormal() compat.AdaptiveColor  { return t.BorderNormalColor }
func (t *BaseTheme) BorderFocused() compat.AdaptiveColor { return t.BorderFocusedColor }
func (t *BaseTheme) BorderDim() compat.AdaptiveColor     { return t.BorderDimColor }

func (t *BaseTheme) DiffAdded() compat.AdaptiveColor            { return t.DiffAddedColor }
func (t *BaseTheme) DiffRemoved() compat.AdaptiveColor          { return t.DiffRemovedColor }
func (t *BaseTheme) DiffContext() compat.AdaptiveColor          { return t.DiffContextColor }
func (t *BaseTheme) DiffHunkHeader() compat.AdaptiveColor       { return t.DiffHunkHeaderColor }
func (t *BaseTheme) DiffHighlightAdded() compat.AdaptiveColor   { return t.DiffHighlightAddedColor }
func (t *BaseTheme) DiffHighlightRemoved() compat.AdaptiveColor { return t.DiffHighlightRemovedColor }
func (t *BaseTheme) DiffAddedBg() compat.AdaptiveColor          { return t.DiffAddedBgColor }
func (t *BaseTheme) DiffRemovedBg() compat.AdaptiveColor        { return t.DiffRemovedBgColor }
func (t *BaseTheme) DiffContextBg() compat.AdaptiveColor        { return t.DiffContextBgColor }
func (t *BaseTheme) DiffLineNumber() compat.AdaptiveColor       { return t.DiffLineNumberColor }
func (t *BaseTheme) DiffAddedLineNumberBg() compat.AdaptiveColor {
	return t.DiffAddedLineNumberBgColor
}
func (t *BaseTheme) DiffRemovedLineNumberBg() compat.AdaptiveColor {
	return t.DiffRemovedLineNumberBgColor
}

func (t *BaseTheme) MarkdownText() compat.AdaptiveColor       { return t.MarkdownTextColor }
func (t *BaseTheme) MarkdownHeading() compat.AdaptiveColor    { return t.MarkdownHeadingColor }
func (t *BaseTheme) MarkdownLink() compat.AdaptiveColor       { return t.MarkdownLinkColor }
func (t *BaseTheme) MarkdownLinkText() compat.AdaptiveColor   { return t.MarkdownLinkTextColor }
func (t *BaseTheme) MarkdownCode() compat.AdaptiveColor       { return t.MarkdownCodeColor }
func (t *BaseTheme) MarkdownBlockQuote() compat.AdaptiveColor { return t.MarkdownBlockQuoteColor }
func (t *BaseTheme) MarkdownEmph() compat.AdaptiveColor       { return t.MarkdownEmphColor }
func (t *BaseTheme) MarkdownStrong() compat.AdaptiveColor     { return t.MarkdownStrongColor }
func (t *BaseTheme) MarkdownHorizontalRule() compat.AdaptiveColor {
	return t.MarkdownHorizontalRuleColor
}
func (t *BaseTheme) MarkdownListItem() compat.AdaptiveColor { return t.MarkdownListItemColor }
func (t *BaseTheme) MarkdownListEnumeration() compat.AdaptiveColor {
	return t.MarkdownListEnumerationColor
}
func (t *BaseTheme) MarkdownImage() compat.AdaptiveColor     { return t.MarkdownImageColor }
func (t *BaseTheme) MarkdownImageText() compat.AdaptiveColor { return t.MarkdownImageTextColor }
func (t *BaseTheme) MarkdownCodeBlock() compat.AdaptiveColor { return t.MarkdownCodeBlockColor }

func (t *BaseTheme) SyntaxComment() compat.AdaptiveColor     { return t.SyntaxCommentColor }
func (t *BaseTheme) SyntaxKeyword() compat.AdaptiveColor     { return t.SyntaxKeywordColor }
func (t *BaseTheme) SyntaxFunction() compat.AdaptiveColor    { return t.SyntaxFunctionColor }
func (t *BaseTheme) SyntaxVariable() compat.AdaptiveColor    { return t.SyntaxVariableColor }
func (t *BaseTheme) SyntaxString() compat.AdaptiveColor      { return t.SyntaxStringColor }
func (t *BaseTheme) SyntaxNumber() compat.AdaptiveColor      { return t.SyntaxNumberColor }
func (t *BaseTheme) SyntaxType() compat.AdaptiveColor        { return t.SyntaxTypeColor }
func (t *BaseTheme) SyntaxOperator() compat.AdaptiveColor    { return t.SyntaxOperatorColor }
func (t *BaseTheme) SyntaxPunctuation() compat.AdaptiveColor { return t.SyntaxPunctuationColor }
