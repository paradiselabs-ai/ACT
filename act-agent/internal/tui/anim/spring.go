// Package anim provides Bubble Tea animation helpers built on harmonica spring
// physics. All animations share a single FrameMsg type so different components
// can coexist without creating duplicate ticker conflicts.
package anim

import (
	"image/color"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/charmbracelet/harmonica"
)

// FPS is the target frame rate for all animations.
const FPS = 60

// FrameMsg is the tick message used to drive animation frames.
type FrameMsg time.Time

// Frame returns a Cmd that fires a FrameMsg after one frame period (1/FPS).
func Frame() tea.Cmd {
	return tea.Tick(time.Second/FPS, func(t time.Time) tea.Msg {
		return FrameMsg(t)
	})
}

// NewSpring creates a harmonica spring tuned for fast, snappy UI transitions.
// stiffness ≈ 8–12 for quick dialogs; damping ≈ 0.5–0.6 for slight overshoot.
func NewSpring(stiffness, damping float64) harmonica.Spring {
	return harmonica.NewSpring(harmonica.FPS(FPS), stiffness, damping)
}

// LerpColor linearly interpolates two color.Color values at parameter t ∈ [0,1].
func LerpColor(from, to color.Color, t float64) color.Color {
	r1, g1, b1, a1 := from.RGBA()
	r2, g2, b2, a2 := to.RGBA()
	lerp := func(a, b uint32) uint8 {
		v := float64(a>>8) + (float64(b>>8)-float64(a>>8))*t
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	return color.RGBA{
		R: lerp(r1, r2),
		G: lerp(g1, g2),
		B: lerp(b1, b2),
		A: lerp(a1, a2),
	}
}

// LerpAdaptive interpolates two AdaptiveColors for both dark and light variants.
func LerpAdaptive(from, to compat.AdaptiveColor, t float64) compat.AdaptiveColor {
	return compat.AdaptiveColor{
		Dark:  LerpColor(from.Dark, to.Dark, t),
		Light: LerpColor(from.Light, to.Light, t),
	}
}
