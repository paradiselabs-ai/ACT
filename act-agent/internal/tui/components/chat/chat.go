package chat

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
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
	baseStyle := styles.BaseStyle()

	lsps := baseStyle.
		Width(width).
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
		lspName := baseStyle.
			Foreground(t.Text()).
			Render(fmt.Sprintf("• %s", name))

		cmd := lsp.Command
		cmd = ansi.Truncate(cmd, width-lipgloss.Width(lspName)-3, "…")

		lspPath := baseStyle.
			Foreground(t.TextMuted()).
			Render(fmt.Sprintf(" (%s)", cmd))

		lspViews = append(lspViews,
			baseStyle.
				Width(width).
				Render(
					lipgloss.JoinHorizontal(
						lipgloss.Left,
						lspName,
						lspPath,
					),
				),
		)
	}

	return baseStyle.
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
	baseStyle := styles.BaseStyle()

	versionText := baseStyle.
		Foreground(t.TextMuted()).
		Render(version.Version)

	return baseStyle.
		Bold(true).
		Width(width).
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
		Width(width).
		Render(repo)
}

func cwd(width int) string {
	cwd := fmt.Sprintf("cwd: %s", config.WorkingDirectory())
	t := theme.CurrentTheme()

	return styles.BaseStyle().
		Foreground(t.TextMuted()).
		Width(width).
		Render(cwd)
}

// actBanner renders the AGENT / COORDINATION / TOOLKIT stacked banner,
// colored with the current theme.
func actBanner(width int) string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	cwdLine := baseStyle.Foreground(t.TextMuted()).Render(
		fmt.Sprintf("  cwd: %s", config.WorkingDirectory()),
	)

	// Narrow terminal fallback — full art is exactly 94 cols wide.
	if width < bannerWidth {
		vStr := ""
		if version.Version != "" {
			vStr = "  " + baseStyle.Foreground(t.TextMuted()).Render(version.Version)
		}
		line := baseStyle.Bold(true).Foreground(t.Primary()).
			Render(styles.ACTIcon + " ACT — Agent Coordination Toolkit")
		return lipgloss.JoinVertical(lipgloss.Left, line+vStr, "", cwdLine)
	}

	primary := baseStyle.Foreground(t.Primary()).Bold(true)
	secondary := baseStyle.Foreground(t.Secondary()).Bold(true)
	accent := baseStyle.Foreground(t.Accent()).Bold(true)
	muted := baseStyle.Foreground(t.TextMuted())

	var b strings.Builder
	for _, l := range agentLines {
		b.WriteString(primary.Render(l) + "\n")
	}
	for _, l := range coordinationLines {
		b.WriteString(secondary.Render(l) + "\n")
	}
	for _, l := range toolkitLines {
		b.WriteString(accent.Render(l) + "\n")
	}
	b.WriteString("\n")

	tag1 := "nested TTY for multi-agent coordination"
	tag2 := "Planner · Observer · Assurance · QA"
	if version.Version != "" {
		tag2 += "  ·  v" + version.Version
	}

	b.WriteString(muted.Render(strings.Repeat("─", bannerWidth)) + "\n")
	b.WriteString(muted.Render(centerLine(tag1, bannerWidth)) + "\n")
	b.WriteString(muted.Render(centerLine(tag2, bannerWidth)) + "\n")

	return lipgloss.JoinVertical(lipgloss.Left, b.String(), cwdLine)
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
func welcomeGuide(width int) string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	sectionTitle := func(text string) string {
		return baseStyle.Foreground(t.Primary()).Bold(true).Render("  " + text)
	}

	muted := func(text string) string {
		return baseStyle.Foreground(t.TextMuted()).Render("    " + text)
	}

	keyStyle := baseStyle.Foreground(t.Text()).Bold(true)
	descStyle := baseStyle.Foreground(t.TextMuted())
	cmdStyle := baseStyle.Foreground(t.Accent())

	shortcut := func(key, desc string) string {
		return "    " + keyStyle.Width(14).Render(key) + descStyle.Render(desc)
	}

	command := func(name, desc string) string {
		return "    " + cmdStyle.Width(18).Render(name) + descStyle.Render(desc)
	}

	sepLen := min(36, width-6)
	if sepLen < 1 {
		sepLen = 1
	}
	sep := baseStyle.Foreground(t.BorderDim()).Render("  " + strings.Repeat("·", sepLen))

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

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
