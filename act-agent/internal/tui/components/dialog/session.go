package dialog

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

// SessionSelectedMsg is sent when a session is selected
type SessionSelectedMsg struct {
	Session session.Session
}

// CloseSessionDialogMsg is sent when the session dialog is closed
type CloseSessionDialogMsg struct{}

// SessionDialog interface for the session switching dialog
type SessionDialog interface {
	tea.Model
	layout.Bindings
	SetSessions(sessions []session.Session)
	SetSelectedSession(sessionID string)
}

type sessionDialogCmp struct {
	form              *huh.Form
	sessions          []session.Session
	selected          session.Session
	selectedSessionID string
}

func (s *sessionDialogCmp) buildForm() {
	opts := make([]huh.Option[session.Session], len(s.sessions))
	for i, sess := range s.sessions {
		label := sess.Title
		if sess.ID == s.selectedSessionID {
			label = "▶ " + label
		}
		opts[i] = huh.NewOption(label, sess)
	}
	s.form = huh.NewForm(huh.NewGroup(
		huh.NewSelect[session.Session]().
			Title("Switch Session").
			Height(10).
			Options(opts...).
			Value(&s.selected),
	)).WithShowHelp(false).WithTheme(actHuhTheme()).WithWidth(48)
	_ = s.form.Init()
}

func (s *sessionDialogCmp) Init() tea.Cmd {
	return nil
}

func (s *sessionDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.form == nil {
		return s, nil
	}
	m, cmd := s.form.Update(msg)
	if f, ok := m.(*huh.Form); ok {
		s.form = f
	}
	switch s.form.State {
	case huh.StateCompleted:
		return s, util.CmdHandler(SessionSelectedMsg{Session: s.selected})
	case huh.StateAborted:
		return s, util.CmdHandler(CloseSessionDialogMsg{})
	}
	return s, cmd
}

func (s *sessionDialogCmp) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	if s.form == nil || len(s.sessions) == 0 {
		return tea.NewView(baseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(t.Background()).
			BorderForeground(t.BorderFocused()).
			Width(40).
			Render("No sessions available"))
	}
	return tea.NewView(baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.BorderFocused()).
		Width(54).
		Render(s.form.View()))
}

func (s *sessionDialogCmp) BindingKeys() []key.Binding {
	if s.form == nil {
		return nil
	}
	return s.form.KeyBinds()
}

func (s *sessionDialogCmp) SetSessions(sessions []session.Session) {
	s.sessions = sessions
	s.selected = session.Session{}
	s.buildForm()
}

func (s *sessionDialogCmp) SetSelectedSession(sessionID string) {
	s.selectedSessionID = sessionID
	if s.form != nil {
		s.buildForm()
	}
}

// NewSessionDialogCmp creates a new session switching dialog
func NewSessionDialogCmp() SessionDialog {
	return &sessionDialogCmp{}
}
