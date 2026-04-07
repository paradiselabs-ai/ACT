package chat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	logo := fmt.Sprintf("%s %s", styles.ACTIcon, "ACT Agent")
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

// actBanner renders a gradient-colored ASCII art banner inside a rounded border.
func actBanner(width int) string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	// Narrow terminal fallback
	if width < 50 {
		vStr := ""
		if version.Version != "" {
			vStr = "  " + baseStyle.Foreground(t.TextMuted()).Render(version.Version)
		}
		return baseStyle.Bold(true).Foreground(t.Primary()).Width(width).
			Render(styles.ACTIcon+" ACT — Agent Coordination Toolkit"+vStr) + "\n" + cwd(width)
	}

	// Block letter ASCII art
	artLines := []string{
		`     █████╗  ██████╗████████╗`,
		`    ██╔══██╗██╔════╝╚══██╔══╝`,
		`    ███████║██║        ██║`,
		`    ██╔══██║██║        ██║`,
		`    ██║  ██║╚██████╗   ██║`,
		`    ╚═╝  ╚═╝ ╚═════╝   ╚═╝`,
	}

	// Gradient colors: Primary → Secondary → Accent (top to bottom)
	gradientColors := []lipgloss.TerminalColor{
		t.Primary(),
		t.Primary(),
		t.Secondary(),
		t.Secondary(),
		t.Accent(),
		t.Accent(),
	}

	// Render each line with its gradient color
	var renderedLines []string
	for i, line := range artLines {
		color := gradientColors[i]
		renderedLines = append(renderedLines,
			baseStyle.Foreground(color).Bold(true).Render(line),
		)
	}

	artBlock := strings.Join(renderedLines, "\n")

	// Subtitle line
	vStr := ""
	if version.Version != "" {
		vStr = "  " + baseStyle.Foreground(t.TextMuted()).Render(version.Version)
	}
	subtitle := "    " + baseStyle.Foreground(t.Text()).Bold(true).
		Render(styles.ACTIcon+" Agent Coordination Toolkit") + vStr

	// Combine art + subtitle inside a rounded border
	innerContent := lipgloss.JoinVertical(lipgloss.Left,
		"",
		artBlock,
		"",
		subtitle,
		"",
	)

	boxWidth := 48
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	bordered := baseStyle.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary()).
		Width(boxWidth).
		Padding(0, 1).
		Render(innerContent)

	// cwd outside the box
	cwdLine := baseStyle.Foreground(t.TextMuted()).Render(
		fmt.Sprintf("  cwd: %s", config.WorkingDirectory()),
	)

	return lipgloss.JoinVertical(lipgloss.Left, bordered, "", cwdLine)
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
		command("act:status", "Coordination server status"),
		command("act:tasks", "Task queue and progress"),
		command("act:agents", "List registered agents"),
		command("act:log", "Recent coordination log"),
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

