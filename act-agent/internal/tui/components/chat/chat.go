package chat

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/charmbracelet/x/ansi"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/anim"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/version"
)

type SendMsg struct {
	Text        string
	Attachments []message.Attachment
}

type SessionSelectedMsg = session.Session

type SessionClearedMsg struct{}

type EditorFocusMsg bool

// ScrollFocusMsg is broadcast when the user toggles scroll-focus mode (Tab).
// When On=true the message viewport captures arrow keys; when false the editor
// regains normal input.
type ScrollFocusMsg struct{ On bool }

// ScrollMsg is broadcast by chatPage when scroll-focus is active and the user
// presses Up/Down. messagesCmp handles it and moves the viewport.
type ScrollMsg struct{ Lines int } // negative = up, positive = down

func header(width int) string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		logo(width),
		repo(width),
		"",
		cwd(width),
	)
}

func lspsConfigured(width int) string {
	cfg := config.Get()
	title := "LSP Configuration"
	title = ansi.Truncate(title, width, "…")

	t := theme.CurrentTheme()
	plain := styles.BaseStyle()

	lsps := plain.
		Foreground(t.Primary()).
		Bold(true).
		Render(title)

	// Get LSP names and sort them for consistent ordering
	var lspNames []string
	for name := range cfg.LSP {
		lspNames = append(lspNames, name)
	}
	sort.Strings(lspNames)

	var lspViews []string
	for _, name := range lspNames {
		lsp := cfg.LSP[name]
		lspName := plain.
			Foreground(t.Text()).
			Render(fmt.Sprintf("• %s", name))

		cmd := lsp.Command
		cmd = ansi.Truncate(cmd, width-lipgloss.Width(lspName)-3, "…")

		lspPath := plain.
			Foreground(t.TextMuted()).
			Render(fmt.Sprintf(" (%s)", cmd))

		lspViews = append(lspViews,
			plain.
				Render(
					lipgloss.JoinHorizontal(
						lipgloss.Left,
						lspName,
						lspPath,
					),
				),
			)
	}

	return styles.BaseStyle().
		Width(width).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				lsps,
				lipgloss.JoinVertical(
					lipgloss.Left,
					lspViews...,
				),
			),
		)
}

func logo(width int) string {
	logo := fmt.Sprintf("%s %s", styles.ACTIcon, "ACT")
	t := theme.CurrentTheme()
	plain := styles.BaseStyle()

	versionText := plain.
		Foreground(t.TextMuted()).
		Render(version.Version)

	return plain.
		Bold(true).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				logo,
				" ",
				versionText,
			),
		)
}

func repo(width int) string {
	repo := "ACT — Agent Coordination Toolkit"
	t := theme.CurrentTheme()

	return styles.BaseStyle().
		Foreground(t.TextMuted()).
		Render(repo)
}

func cwd(width int) string {
	cwd := fmt.Sprintf("cwd: %s", config.WorkingDirectory())
	t := theme.CurrentTheme()

	return styles.BaseStyle().
		Foreground(t.TextMuted()).
		Render(cwd)
}

// actBanner renders the AGENT / COORDINATION / TOOLKIT stacked banner,
// colored with the current theme. alpha ∈ [0,1] fades colors from muted→full.
func actBanner(width int, alpha float64) string {
	t := theme.CurrentTheme()
	plain := styles.BaseStyle()
	row := func(s string) string { return plain.Width(width).Render(s) }

	// Lerp helper: blend Background → target color at the given alpha.
	// Starting from Background makes the text emerge from invisible → full color.
	fadeColor := func(target compat.AdaptiveColor) compat.AdaptiveColor {
		if alpha >= 1.0 {
			return target
		}
		return anim.LerpAdaptive(t.Background(), target, alpha)
	}

	cwdLine := plain.Foreground(fadeColor(t.TextMuted())).Render(
		fmt.Sprintf("  cwd: %s", config.WorkingDirectory()),
	)

	// Narrow terminal fallback — full art is exactly 94 cols wide.
	if width < bannerWidth {
		vStr := ""
		if version.Version != "" {
			vStr = plain.Foreground(fadeColor(t.TextMuted())).Render("  " + version.Version)
		}
		line := plain.Bold(true).Foreground(fadeColor(t.Primary())).
			Render(styles.ACTIcon + " ACT — Agent Coordination Toolkit")
		return strings.Join([]string{
			row(line + vStr),
			row(""),
			row(cwdLine),
		}, "\n")
	}

	primary := plain.Foreground(fadeColor(t.Primary())).Bold(true)
	secondary := plain.Foreground(fadeColor(t.Secondary())).Bold(true)
	accent := plain.Foreground(fadeColor(t.Accent())).Bold(true)
	muted := plain.Foreground(fadeColor(t.TextMuted()))

	var b strings.Builder
	for _, l := range agentLines {
		b.WriteString(row(primary.Render(l)) + "\n")
	}
	for _, l := range coordinationLines {
		b.WriteString(row(secondary.Render(l)) + "\n")
	}
	for _, l := range toolkitLines {
		b.WriteString(row(accent.Render(l)) + "\n")
	}
	b.WriteString(row("") + "\n")

	tag1 := "nested TTY for multi-agent coordination"
	tag2 := "Planner · Observer · Assurance · QA"
	if version.Version != "" {
		tag2 += "  ·  v" + version.Version
	}

	b.WriteString(row(muted.Render(strings.Repeat("─", bannerWidth))) + "\n")
	b.WriteString(row(muted.Render(centerLine(tag1, bannerWidth))) + "\n")
	b.WriteString(row(muted.Render(centerLine(tag2, bannerWidth))) + "\n")

	return b.String() + row(cwdLine)
}

func centerLine(s string, w int) string {
	pad := (w - len([]rune(s))) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + s
}

// ── pre-rendered ansi_shadow art ─────────────────────────────────────────────

const bannerWidth = 94

var agentLines = []string{
	`                          █████╗  ██████╗ ███████╗███╗   ██╗████████╗`,
	`                         ██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝`,
	`                         ███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║`,
	`                         ██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║`,
	`                         ██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║`,
	`                         ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝`,
}

var coordinationLines = []string{
	` ██████╗ ██████╗  ██████╗ ██████╗ ██████╗ ██╗███╗   ██╗ █████╗ ████████╗██╗ ██████╗ ███╗   ██╗`,
	`██╔════╝██╔═══██╗██╔═══██╗██╔══██╗██╔══██╗██║████╗  ██║██╔══██╗╚══██╔══╝██║██╔═══██╗████╗  ██║`,
	`██║     ██║   ██║██║   ██║██████╔╝██║  ██║██║██╔██╗ ██║███████║   ██║   ██║██║   ██║██╔██╗ ██║`,
	`██║     ██║   ██║██║   ██║██╔══██╗██║  ██║██║██║╚██╗██║██╔══██║   ██║   ██║██║   ██║██║╚██╗██║`,
	`╚██████╗╚██████╔╝╚██████╔╝██║  ██║██████╔╝██║██║ ╚████║██║  ██║   ██║   ██║╚██████╔╝██║ ╚████║`,
	` ╚═════╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═════╝ ╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝   ╚═╝   ╚═╝ ╚═════╝ ╚═╝  ╚═══╝`,
}

var toolkitLines = []string{
	`                   ████████╗ ██████╗  ██████╗ ██╗     ██╗  ██╗██╗████████╗`,
	`                   ╚══██╔══╝██╔═══██╗██╔═══██╗██║     ██║ ██╔╝██║╚══██╔══╝`,
	`                      ██║   ██║   ██║██║   ██║██║     █████╔╝ ██║   ██║`,
	`                      ██║   ██║   ██║██║   ██║██║     ██╔═██╗ ██║   ██║`,
	`                      ██║   ╚██████╔╝╚██████╔╝███████╗██║  ██╗██║   ██║`,
	`                      ╚═╝    ╚═════╝  ╚═════╝ ╚══════╝╚═╝  ╚═╝╚═╝   ╚═╝`,
}

// welcomeGuide renders a quick-start reference for the initial screen.
// alpha ∈ [0,1] is passed through from the splash fade-in spring.
func welcomeGuide(width int, alpha float64) string {
	t := theme.CurrentTheme()
	plain := styles.BaseStyle()

	fadeColor := func(target compat.AdaptiveColor) compat.AdaptiveColor {
		if alpha >= 1.0 {
			return target
		}
		return anim.LerpAdaptive(t.Background(), target, alpha)
	}

	sectionTitle := func(text string) string {
		return plain.Foreground(fadeColor(t.Primary())).Bold(true).Render("  " + text)
	}

	muted := func(text string) string {
		return plain.Foreground(fadeColor(t.TextMuted())).Render("    " + text)
	}

	keyStyle := plain.Foreground(fadeColor(t.Text())).Bold(true)
	descStyle := plain.Foreground(fadeColor(t.TextMuted()))
	cmdStyle := plain.Foreground(fadeColor(t.Accent()))

	shortcut := func(key, desc string) string {
		return "    " + keyStyle.Width(14).Render(key) + descStyle.Render(desc)
	}

	command := func(name, desc string) string {
		return "    " + cmdStyle.Width(24).Render(name) + descStyle.Render(desc)
	}

	sepLen := min(36, width-6)
	if sepLen < 1 {
		sepLen = 1
	}
	sep := plain.Foreground(fadeColor(t.BorderDim())).Render("  " + strings.Repeat("·", sepLen))

	lines := []string{
		sectionTitle("Getting Started"),
		muted("Type a message below to start a conversation."),
		muted("ACT coordinates multiple AI agents across your project."),
		"",
		sep,
		"",
		sectionTitle("Quick Reference"),
		shortcut("ctrl+k", "Command palette"),
		shortcut("ctrl+s", "Switch session"),
		shortcut("ctrl+?", "All keybindings"),
		"",
		sep,
		"",
		sectionTitle("Commands  (ctrl+k)"),
		command("init", "Create ACT.md project memory"),
		command("act-agent:status", "Server, agents, projects"),
		command("act-agent:log", "Recent coordination log"),
		command("act-agent:tasks", "Tasks awaiting validation"),
		command("act-agent:validation", "Assurance queue"),
		command("act-agent:conflicts", "File lock conflicts"),
		command("act-agent:swarm", "Per-role backend"),
	}

	return styles.BaseStyle().Width(width).Render(strings.Join(lines, "\n"))
}
