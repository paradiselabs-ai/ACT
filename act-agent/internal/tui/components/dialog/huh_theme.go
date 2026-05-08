package dialog

import (
	"charm.land/huh/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
)

// actHuhTheme returns a huh.Theme that mirrors the current ACT theme palette.
// Patches ThemeCharm so selector, title, and button styles use the same
// Primary/TextMuted/Background colors as the rest of the TUI, regardless of
// which of the 9 built-in themes is active.
//
// Call this inside every form builder so that theme-switches (ThemeChangedMsg)
// are picked up on the next form rebuild.
func actHuhTheme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		t := theme.CurrentTheme()
		s := huh.ThemeCharm(isDark)

		// Top-level form and group wrappers
		s.Form.Base = s.Form.Base.Background(t.Background())
		s.Group.Base = s.Group.Base.Background(t.Background())
		s.FieldSeparator = s.FieldSeparator.Background(t.Background())

		// Focused field: primary accent on title, selector cursor, and selection
		s.Focused.Base = s.Focused.Base.BorderForeground(t.Primary()).Background(t.Background())
		s.Focused.Title = s.Focused.Title.Foreground(t.Primary()).Background(t.Background())
		s.Focused.SelectSelector = s.Focused.SelectSelector.Foreground(t.Primary()).Background(t.Background())
		s.Focused.SelectedOption = s.Focused.SelectedOption.Foreground(t.Primary()).Background(t.Background())
		s.Focused.Option = s.Focused.Option.Background(t.Background())
		s.Focused.UnselectedOption = s.Focused.UnselectedOption.Background(t.Background())

		// Confirm buttons: focused = inverted primary, blurred = muted
		s.Focused.FocusedButton = s.Focused.FocusedButton.
			Background(t.Primary()).
			Foreground(t.Background())
		s.Focused.BlurredButton = s.Focused.BlurredButton.
			Foreground(t.TextMuted()).
			Background(t.Background())

		// Blurred field: dim border and muted title
		s.Blurred.Base = s.Blurred.Base.BorderForeground(t.BorderDim()).Background(t.Background())
		s.Blurred.Title = s.Blurred.Title.Foreground(t.TextMuted()).Background(t.Background())
		s.Blurred.Option = s.Blurred.Option.Background(t.Background())
		s.Blurred.UnselectedOption = s.Blurred.UnselectedOption.Background(t.Background())

		return s
	})
}
