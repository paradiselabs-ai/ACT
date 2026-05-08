package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
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
	info       util.InfoMsg
	width      int
	messageTTL time.Duration
	lspClients map[string]*lsp.Client
	session    session.Session
}

// clearMessageCmd is a command that clears status messages after a timeout
func (m statusCmp) clearMessageCmd(ttl time.Duration) tea.Cmd {
	return tea.Tick(ttl, func(time.Time) tea.Msg {
		return util.ClearStatusMsg{}
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
		ttl := msg.TTL
		if ttl == 0 {
			ttl = m.messageTTL
		}
		return m, m.clearMessageCmd(ttl)
	case util.ClearStatusMsg:
		m.info = util.InfoMsg{}
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

	// For session token tracking, use the largest context window across all
	// Tier 1 agents — the conversation as a whole is bounded by whichever
	// agent has the most headroom.
	var contextWindow int64
	for _, cfg := range config.Tier1Configs() {
		if w := models.SupportedModels[cfg.Model].ContextWindow; w > contextWindow {
			contextWindow = w
		}
	}

	// Initialize the help widget
	status := getHelpWidget()

	tokenInfoWidth := 0
	if m.session.ID != "" && contextWindow > 0 {
		totalTokens := m.session.PromptTokens + m.session.CompletionTokens
		tokens := formatTokensAndCost(totalTokens, contextWindow, m.session.Cost)
		tokensStyle := styles.Padded().
			Background(t.Text()).
			Foreground(t.BackgroundSecondary())
		percentage := (float64(totalTokens) / float64(contextWindow)) * 100
		if percentage > 80 {
			tokensStyle = tokensStyle.Background(t.Warning())
		}
		tokenInfoWidth = lipgloss.Width(tokens) + 2
		status += tokensStyle.Render(tokens)
	}

	diagnostics := styles.Padded().
		Background(t.BackgroundDarker()).
		Render(m.projectDiagnostics())

	tier1 := m.tier1Models()
	availableWidht := max(0, m.width-lipgloss.Width(helpWidget)-lipgloss.Width(tier1)-lipgloss.Width(diagnostics)-tokenInfoWidth)

	// Status bar minimum: when the right-side widgets eat almost everything,
	// we used to render the message into a 5-char-wide cell which forced
	// lipgloss to wrap a 600-char OpenRouter error into an unreadable
	// vertical waterfall. Two safety rails fix this:
	//
	//   1. Collapse `tier1Models` to a tiny "P/O/A/Q" glyph if there isn't
	//      enough room left for a readable message. We never silently kill
	//      the message — the persistent widgets yield first.
	//   2. Truncate the message with ANSI-aware single-line truncation
	//      BEFORE handing it to lipgloss, regardless of computed widths.
	//      The truncation always runs; the only question is what width.
	const minMsgRenderWidth = 24
	if m.info.Msg != "" && availableWidht < minMsgRenderWidth {
		// Drop the verbose tier1 widget for one render and recompute.
		// This is a one-frame collapse — next non-error frame restores it.
		tier1 = m.tier1ModelsCompact()
		availableWidht = max(0, m.width-lipgloss.Width(helpWidget)-lipgloss.Width(tier1)-lipgloss.Width(diagnostics)-tokenInfoWidth)
	}

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
		status += styles.Padded().
			Foreground(t.Text()).
			Background(t.BackgroundSecondary()).
			Width(availableWidht).
			Render("")
	}

	status += diagnostics
	status += tier1
	return tea.NewView(status)
}

// tier1ModelsCompact renders the Tier 1 model strip in its tightest form
// (P/O/A/Q glyphs only, no model names) so the status bar can yield
// horizontal space to a long message without dropping the widget entirely.
func (m statusCmp) tier1ModelsCompact() string {
	t := theme.CurrentTheme()
	return styles.Padded().
		Background(t.Secondary()).
		Foreground(t.Background()).
		Render("P/O/A/Q")
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

func (m statusCmp) availableFooterMsgWidth(diagnostics, tokenInfo string) int {
	tokensWidth := 0
	if m.session.ID != "" {
		tokensWidth = lipgloss.Width(tokenInfo) + 2
	}
	return max(0, m.width-lipgloss.Width(helpWidget)-lipgloss.Width(m.tier1Models())-lipgloss.Width(diagnostics)-tokensWidth)
}

// tier1Models renders a compact display of all four Tier 1 agents and their
// configured models in the form "P:Opus O:Sonnet A:Sonnet Q:Sonnet". There
// is no "default" agent in ACT — the four NesTTY agents share the conversation
// and each has its own LLM. The status bar reflects that reality.
func (m statusCmp) tier1Models() string {
	t := theme.CurrentTheme()
	tier1 := config.Tier1Configs()

	parts := make([]string, 0, 4)
	for _, name := range config.Tier1AgentNames() {
		cfg, ok := tier1[name]
		label := config.Tier1ShortLabel(name)
		if !ok || cfg.Model == "" {
			parts = append(parts, label+":-")
			continue
		}
		modelName := models.SupportedModels[cfg.Model].Name
		if modelName == "" {
			modelName = string(cfg.Model)
		}
		// Trim long model names so the status bar doesn't blow out
		if len(modelName) > 12 {
			modelName = modelName[:12]
		}
		parts = append(parts, label+":"+modelName)
	}

	return styles.Padded().
		Background(t.Secondary()).
		Foreground(t.Background()).
		Render(strings.Join(parts, " "))
}

func NewStatusCmp(lspClients map[string]*lsp.Client) StatusCmp {
	helpWidget = getHelpWidget()

	return &statusCmp{
		messageTTL: 10 * time.Second,
		lspClients: lspClients,
	}
}
