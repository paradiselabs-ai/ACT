package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/app"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/lsp"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/lsp/protocol"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/chat"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

type StatusCmp interface {
	tea.Model
}

type statusCmp struct {
	app        *app.App
	info       util.InfoMsg
	width      int
	messageTTL time.Duration
	lspClients map[string]*lsp.Client
	session    session.Session

	// infoGen increments on every InfoMsg. Each clear timer is bound to the
	// generation it was scheduled for; a stale tick (older gen) is ignored.
	// Without this, two messages 5s apart meant the FIRST message's 10s tick
	// cleared the SECOND message halfway through its lifetime.
	infoGen uint64
}

// clearMessageCmd is a command that clears status messages after a timeout,
// scoped to the generation that scheduled it
func (m statusCmp) clearMessageCmd(ttl time.Duration, gen uint64) tea.Cmd {
	return tea.Tick(ttl, func(time.Time) tea.Msg {
		return util.ClearStatusMsg{Gen: gen}
	})
}

func (m statusCmp) Init() tea.Cmd {
	return nil
}

func (m statusCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case chat.SessionSelectedMsg:
		m.session = msg
	case chat.SessionClearedMsg:
		m.session = session.Session{}
	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.UpdatedEvent {
			if m.session.ID == msg.Payload.ID {
				m.session = msg.Payload
			}
		}
	case util.InfoMsg:
		m.info = msg
		m.infoGen++
		ttl := msg.TTL
		if ttl == 0 {
			ttl = m.messageTTL
		}
		return m, m.clearMessageCmd(ttl, m.infoGen)
	case util.ClearStatusMsg:
		// Ignore ticks scheduled for an older message — only the newest
		// generation may clear the banner.
		if msg.Gen == m.infoGen {
			m.info = util.InfoMsg{}
		}
	}
	return m, nil
}

var helpWidget = ""

// getHelpWidget returns the help widget with current theme colors
func getHelpWidget() string {
	t := theme.CurrentTheme()
	helpText := "ctrl+? help"

	return styles.Padded().
		Background(t.TextMuted()).
		Foreground(t.BackgroundDarker()).
		Bold(true).
		Render(helpText)
}

func formatTokensAndCost(tokens, contextWindow int64, cost float64) string {
	// Format tokens in human-readable format (e.g., 110K, 1.2M)
	var formattedTokens string
	switch {
	case tokens >= 1_000_000:
		formattedTokens = fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	case tokens >= 1_000:
		formattedTokens = fmt.Sprintf("%.1fK", float64(tokens)/1_000)
	default:
		formattedTokens = fmt.Sprintf("%d", tokens)
	}

	// Remove .0 suffix if present
	if strings.HasSuffix(formattedTokens, ".0K") {
		formattedTokens = strings.Replace(formattedTokens, ".0K", "K", 1)
	}
	if strings.HasSuffix(formattedTokens, ".0M") {
		formattedTokens = strings.Replace(formattedTokens, ".0M", "M", 1)
	}

	formattedCost := fmt.Sprintf("$%.2f", cost)

	if contextWindow <= 0 {
		return fmt.Sprintf("%s %s", formattedTokens, formattedCost)
	}

	percentage := (float64(tokens) / float64(contextWindow)) * 100

	// 6-char fill bar: instant at-a-glance context consumption indicator.
	// Uses Unicode block characters that work in any modern terminal.
	const barWidth = 6
	filled := barWidth * int(min(percentage, 100.0)) / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	if percentage > 80 {
		formattedTokens = fmt.Sprintf("%s(%d%%)", styles.WarningIcon, int(percentage))
	}

	return fmt.Sprintf("%s %s %s", bar, formattedTokens, formattedCost)
}

func (m statusCmp) View() tea.View {
	t := theme.CurrentTheme()

	// Context window is no longer tracked statically (model registry
	// removed). The token-percentage bar would need a per-provider
	// /v1/models lookup or user-supplied limit; until that lands, we
	// just hide the bar.
	var contextWindow int64

	// Initialize the help widget
	status := getHelpWidget()

	tokenInfoWidth := 0
	if m.session.ID != "" {
		totalTokens := m.session.PromptTokens + m.session.CompletionTokens
		tokens := formatTokensAndCost(totalTokens, contextWindow, m.session.Cost)
		tokensStyle := styles.Padded().
			Background(t.Text()).
			Foreground(t.BackgroundSecondary())
		if contextWindow > 0 {
			percentage := (float64(totalTokens) / float64(contextWindow)) * 100
			if percentage > 80 {
				tokensStyle = tokensStyle.Background(t.Warning())
			}
		}
		tokenInfoWidth = lipgloss.Width(tokens) + 2
		status += tokensStyle.Render(tokens)
	}

	diagnostics := styles.Padded().
		Background(t.Background()).
		Foreground(t.TextMuted()).
		Render(m.projectDiagnostics())

	availableWidht := max(0, m.width-lipgloss.Width(helpWidget)-lipgloss.Width(diagnostics)-tokenInfoWidth-4)

	if m.info.Msg != "" {
		infoStyle := styles.Padded().
			Foreground(t.Background()).
			Width(availableWidht).
			MaxWidth(availableWidht)

		switch m.info.Type {
		case util.InfoTypeInfo:
			infoStyle = infoStyle.Background(t.Info())
		case util.InfoTypeWarn:
			infoStyle = infoStyle.Background(t.Warning())
		case util.InfoTypeError:
			infoStyle = infoStyle.Background(t.Error())
		}

		// Always collapse newlines first — multi-line errors in a single-line
		// status bar are exactly the wrap-disaster we're fixing.
		msg := strings.ReplaceAll(m.info.Msg, "\n", " ")
		msg = strings.ReplaceAll(msg, "\r", " ")
		// Padded() adds 2 chars of horizontal padding; budget for it plus
		// the ellipsis itself so the truncate doesn't overflow into wrap.
		budget := availableWidht - 4
		if budget < 1 {
			// If even the truncated message can't fit, just show an icon.
			msg = "!"
		} else {
			msg = ansi.Truncate(msg, budget, "...")
		}
		status += infoStyle.Render(msg)
	} else {
		emptyStyle := lipgloss.NewStyle().Background(t.Background())
		status += emptyStyle.Render(strings.Repeat(" ", availableWidht))
	}

	divider := lipgloss.NewStyle().Background(t.Background()).Foreground(t.BorderDim()).Render(" │ ")
	status += divider + diagnostics

	baseStyle := lipgloss.NewStyle().
		Width(m.width).
		Background(t.Background())

	return tea.NewView(baseStyle.Render(status))
}

func getBadge(modelStr string) string {
	// Delegates to the shared styles.ModelBadge — the status strip's private
	// copy and the navigator's diverged (audit H12). Empty model now yields
	// "-" instead of the old misleading "H3" default.
	return styles.ModelBadge(modelStr)
}

// tier1ModelsCompact renders the Tier 1 model strip in its tightest form
// (P/O/A/Q glyphs only, no model names) so the status bar can yield
// horizontal space to a long message without dropping the widget entirely.
func (m statusCmp) tier1ModelsCompact() string {
	t := theme.CurrentTheme()
	phase := app.PhaseIdle
	if m.app != nil && m.app.Orchestrator != nil {
		phase = m.app.Orchestrator.CurrentPhase()
	}

	parts := make([]string, 0, 4)
	for _, name := range config.Tier1AgentNames() {
		label := config.Tier1ShortLabel(name)
		c := styles.AgentColor(string(name))

		state := app.AgentStateIdle
		if m.app != nil && m.app.Orchestrator != nil {
			state = m.app.Orchestrator.AgentState(string(name), phase)
		}

		switch state {
		case app.AgentStateActive:
			parts = append(parts, lipgloss.NewStyle().Foreground(c).Bold(true).Render(label))
		case app.AgentStateWaiting:
			parts = append(parts, lipgloss.NewStyle().Foreground(t.Warning()).Render(label))
		case app.AgentStateFailed:
			parts = append(parts, lipgloss.NewStyle().Foreground(t.Error()).Bold(true).Render(label))
		default:
			parts = append(parts, lipgloss.NewStyle().Foreground(t.TextMuted()).Render(label))
		}
	}

	return styles.Padded().
		Background(t.Background()).
		Render(strings.Join(parts, "/"))
}

func (m *statusCmp) projectDiagnostics() string {
	t := theme.CurrentTheme()

	// Check if any LSP server is still initializing
	initializing := false
	for _, client := range m.lspClients {
		if client.GetServerState() == lsp.StateStarting {
			initializing = true
			break
		}
	}

	// If any server is initializing, show that status
	if initializing {
		return lipgloss.NewStyle().
			Background(t.BackgroundDarker()).
			Foreground(t.Warning()).
			Render(fmt.Sprintf("%s Initializing LSP...", styles.SpinnerIcon))
	}

	errorDiagnostics := []protocol.Diagnostic{}
	warnDiagnostics := []protocol.Diagnostic{}
	hintDiagnostics := []protocol.Diagnostic{}
	infoDiagnostics := []protocol.Diagnostic{}
	for _, client := range m.lspClients {
		for _, d := range client.GetDiagnostics() {
			for _, diag := range d {
				switch diag.Severity {
				case protocol.SeverityError:
					errorDiagnostics = append(errorDiagnostics, diag)
				case protocol.SeverityWarning:
					warnDiagnostics = append(warnDiagnostics, diag)
				case protocol.SeverityHint:
					hintDiagnostics = append(hintDiagnostics, diag)
				case protocol.SeverityInformation:
					infoDiagnostics = append(infoDiagnostics, diag)
				}
			}
		}
	}

	if len(errorDiagnostics) == 0 && len(warnDiagnostics) == 0 && len(hintDiagnostics) == 0 && len(infoDiagnostics) == 0 {
		return "No diagnostics"
	}

	diagnostics := []string{}

	if len(errorDiagnostics) > 0 {
		errStr := lipgloss.NewStyle().
			Background(t.BackgroundDarker()).
			Foreground(t.Error()).
			Render(fmt.Sprintf("%s %d", styles.ErrorIcon, len(errorDiagnostics)))
		diagnostics = append(diagnostics, errStr)
	}
	if len(warnDiagnostics) > 0 {
		warnStr := lipgloss.NewStyle().
			Background(t.BackgroundDarker()).
			Foreground(t.Warning()).
			Render(fmt.Sprintf("%s %d", styles.WarningIcon, len(warnDiagnostics)))
		diagnostics = append(diagnostics, warnStr)
	}
	if len(hintDiagnostics) > 0 {
		hintStr := lipgloss.NewStyle().
			Background(t.BackgroundDarker()).
			Foreground(t.Text()).
			Render(fmt.Sprintf("%s %d", styles.HintIcon, len(hintDiagnostics)))
		diagnostics = append(diagnostics, hintStr)
	}
	if len(infoDiagnostics) > 0 {
		infoStr := lipgloss.NewStyle().
			Background(t.BackgroundDarker()).
			Foreground(t.Info()).
			Render(fmt.Sprintf("%s %d", styles.InfoIcon, len(infoDiagnostics)))
		diagnostics = append(diagnostics, infoStr)
	}

	return strings.Join(diagnostics, " ")
}



// tier1Models renders a compact display of all four Tier 1 agents and their
// configured models in the form "P:Opus O:Sonnet A:Sonnet Q:Sonnet". There
// is no "default" agent in ACT — the four NesTTY agents share the conversation
// and each has its own LLM. The status bar reflects that reality.
func (m statusCmp) tier1Models() string {
	t := theme.CurrentTheme()
	tier1 := config.Tier1Configs()
	phase := app.PhaseIdle
	if m.app != nil && m.app.Orchestrator != nil {
		phase = m.app.Orchestrator.CurrentPhase()
	}

	parts := make([]string, 0, 4)
	for _, name := range config.Tier1AgentNames() {
		cfg, ok := tier1[name]
		label := config.Tier1ShortLabel(name)
		c := styles.AgentColor(string(name))

		state := app.AgentStateIdle
		if m.app != nil && m.app.Orchestrator != nil {
			state = m.app.Orchestrator.AgentState(string(name), phase)
		}

		var styledLabel string
		switch state {
		case app.AgentStateActive:
			styledLabel = lipgloss.NewStyle().Foreground(c).Bold(true).Render(label)
		case app.AgentStateWaiting:
			styledLabel = lipgloss.NewStyle().Foreground(t.Warning()).Render(label)
		case app.AgentStateFailed:
			styledLabel = lipgloss.NewStyle().Foreground(t.Error()).Bold(true).Render(label)
		default:
			styledLabel = lipgloss.NewStyle().Foreground(t.TextMuted()).Render(label)
		}

		if !ok || cfg.Model == "" {
			parts = append(parts, styledLabel+":-")
			continue
		}

		modelName := string(cfg.Model)
		modelBadge := getBadge(modelName)

		var styledValue string
		switch state {
		case app.AgentStateActive:
			styledValue = lipgloss.NewStyle().Foreground(t.Text()).Bold(true).Render(modelBadge)
		case app.AgentStateWaiting:
			styledValue = lipgloss.NewStyle().Foreground(t.TextMuted()).Render(modelBadge)
		default:
			styledValue = lipgloss.NewStyle().Foreground(t.TextMuted()).Render(modelBadge)
		}

		parts = append(parts, styledLabel+":"+styledValue)
	}

	return styles.Padded().
		Background(t.Background()).
		Render(strings.Join(parts, " "))
}

func NewStatusCmp(app *app.App) StatusCmp {
	helpWidget = getHelpWidget()

	return &statusCmp{
		app:        app,
		messageTTL: 10 * time.Second,
		lspClients: app.LSPClients,
	}
}
