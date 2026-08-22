package chat

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/diff"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/history"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
)

type sidebarCmp struct {
	width, height int
	session       session.Session
	history       history.Service
	modFiles      map[string]struct {
		additions int
		removals  int
	}

	// filesCh is subscribed ONCE for the component's lifetime. The previous
	// implementation re-subscribed on every incoming file event, and each
	// subscription's cancel context was context.Background() — never
	// cancelled — so every event permanently leaked a channel + goroutine
	// into the history Broker. The Cmd below re-reads from this same
	// channel instead.
	filesCh   <-chan pubsub.Event[history.File]
	subCancel context.CancelFunc
}

// filesLoadedMsg carries the recomputed modified-files map after an
// asynchronous reload (audit H11: loadModifiedFiles ran two full-session
// DB queries synchronously inside Update on every session switch).
// sessionID guards stale results.
type filesLoadedMsg struct {
	sessionID string
	files     map[string]struct {
		additions int
		removals  int
	}
}

// fileDiffComputedMsg carries the diff stats for one file after an async
// initial-version lookup + diff computation on the event path.
type fileDiffComputedMsg struct {
	sessionID string
	path      string // display path (post workingDir-trim)
	present   bool   // false → remove from map
	additions int
	removals  int
}

func (m *sidebarCmp) Init() tea.Cmd {
	if m.history != nil {
		// Initialize the modified files map
		m.modFiles = make(map[string]struct {
			additions int
			removals  int
		})

		// One subscription, owned by a cancellable context stored on the
		// component so it can be torn down exactly once.
		subCtx, cancel := context.WithCancel(context.Background())
		m.subCancel = cancel
		m.filesCh = m.history.Subscribe(subCtx)

		// Kick the initial load off-loop (audit H11) and start listening.
		return tea.Batch(m.waitForFileEvent(), m.loadModifiedFilesAsync())
	}
	return nil
}

// loadModifiedFilesAsync recomputes the whole modified-files map off the
// Update loop and posts filesLoadedMsg. Snapshot of sessionID is taken at
// construction; the Update handler drops stale results.
func (m *sidebarCmp) loadModifiedFilesAsync() tea.Cmd {
	if m.history == nil || m.session.ID == "" {
		return nil
	}
	sid := m.session.ID
	historySvc := m.history
	workingDir := config.WorkingDirectory()
	return func() tea.Msg {
		files := map[string]struct {
			additions int
			removals  int
		}{}
		ctx := context.Background()

		latestFiles, err := historySvc.ListLatestSessionFiles(ctx, sid)
		if err != nil {
			return filesLoadedMsg{sessionID: sid, files: files}
		}
		allFiles, err := historySvc.ListBySession(ctx, sid)
		if err != nil {
			return filesLoadedMsg{sessionID: sid, files: files}
		}

		initialByPath := make(map[string]history.File)
		for _, v := range allFiles {
			if v.Version == history.InitialVersion {
				initialByPath[v.Path] = v
			}
		}

		for _, file := range latestFiles {
			if file.Version == history.InitialVersion {
				continue
			}
			initialVersion, ok := initialByPath[file.Path]
			if !ok || initialVersion.Content == file.Content {
				continue
			}
			_, additions, removals := diff.GenerateDiff(initialVersion.Content, file.Content, file.Path)
			if additions == 0 && removals == 0 {
				continue
			}
			displayPath := strings.TrimPrefix(strings.TrimPrefix(file.Path, workingDir), "/")
			files[displayPath] = struct {
				additions int
				removals  int
			}{additions: additions, removals: removals}
		}
		return filesLoadedMsg{sessionID: sid, files: files}
	}
}

// waitForFileEvent returns a Cmd that resolves with the next file-history
// event from the component's single subscription.
func (m *sidebarCmp) waitForFileEvent() tea.Cmd {
	ch := m.filesCh
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *sidebarCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SessionSelectedMsg:
		if msg.ID != m.session.ID {
			m.session = msg
			// Async reload — the two full-session DB queries must not run on
			// the Update loop (audit H11).
			return m, m.loadModifiedFilesAsync()
		}
	case filesLoadedMsg:
		// Stale guard: user switched sessions while this load was in flight.
		if msg.sessionID == m.session.ID {
			m.modFiles = msg.files
		}
	case fileDiffComputedMsg:
		if msg.sessionID != m.session.ID {
			break
		}
		displayPath := getDisplayPath(msg.path)
		if !msg.present || (msg.additions == 0 && msg.removals == 0) {
			delete(m.modFiles, displayPath)
			break
		}
		m.modFiles[displayPath] = struct {
			additions int
			removals  int
		}{additions: msg.additions, removals: msg.removals}
	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.UpdatedEvent {
			if m.session.ID == msg.Payload.ID {
				m.session = msg.Payload
			}
		}
	case pubsub.Event[history.File]:
		if msg.Payload.SessionID == m.session.ID {
			// Diff computation (initial-version lookup + GenerateDiff) is DB
			// + CPU work — off-loop via Cmd, result lands as fileDiffComputedMsg.
			cmd := m.processFileChangesAsync(msg.Payload)
			// Keep listening on the SAME subscription — no re-subscribe.
			return m, tea.Batch(cmd, m.waitForFileEvent())
		}
		// Event for another session: still re-arm so the stream doesn't stall.
		return m, m.waitForFileEvent()
	}
	return m, nil
}

func (m *sidebarCmp) View() tea.View {
	baseStyle := styles.BaseStyle()

	return tea.NewView(baseStyle.
		Width(m.width).
		PaddingLeft(4).
		PaddingRight(2).
		Height(m.height - 1).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				header(m.width),
				" ",
				m.sessionSection(),
				" ",
				lspsConfigured(m.width),
				" ",
				m.modifiedFiles(),
			),
		))
}

func (m *sidebarCmp) sessionSection() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	sessionKey := baseStyle.
		Foreground(t.Primary()).
		Bold(true).
		Render("Session")

	sessionValue := baseStyle.
		Foreground(t.Text()).
		Width(m.width - lipgloss.Width(sessionKey)).
		Render(fmt.Sprintf(": %s", m.session.Title))

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		sessionKey,
		sessionValue,
	)
}

func (m *sidebarCmp) modifiedFile(filePath string, additions, removals int) string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	stats := ""
	if additions > 0 && removals > 0 {
		additionsStr := baseStyle.
			Foreground(t.Success()).
			PaddingLeft(1).
			Render(fmt.Sprintf("+%d", additions))

		removalsStr := baseStyle.
			Foreground(t.Error()).
			PaddingLeft(1).
			Render(fmt.Sprintf("-%d", removals))

		content := lipgloss.JoinHorizontal(lipgloss.Left, additionsStr, removalsStr)
		stats = baseStyle.Width(lipgloss.Width(content)).Render(content)
	} else if additions > 0 {
		additionsStr := fmt.Sprintf(" %s", baseStyle.
			PaddingLeft(1).
			Foreground(t.Success()).
			Render(fmt.Sprintf("+%d", additions)))
		stats = baseStyle.Width(lipgloss.Width(additionsStr)).Render(additionsStr)
	} else if removals > 0 {
		removalsStr := fmt.Sprintf(" %s", baseStyle.
			PaddingLeft(1).
			Foreground(t.Error()).
			Render(fmt.Sprintf("-%d", removals)))
		stats = baseStyle.Width(lipgloss.Width(removalsStr)).Render(removalsStr)
	}

	filePathStr := baseStyle.Render(filePath)

	return baseStyle.
		Width(m.width).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				filePathStr,
				stats,
			),
		)
}

func (m *sidebarCmp) modifiedFiles() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	modifiedFiles := baseStyle.
		Width(m.width).
		Foreground(t.Primary()).
		Bold(true).
		Render("Modified Files:")

	// If no modified files, show a placeholder message
	if len(m.modFiles) == 0 {
		message := "No modified files"
		remainingWidth := m.width - lipgloss.Width(message)
		if remainingWidth > 0 {
			message += strings.Repeat(" ", remainingWidth)
		}
		return baseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					modifiedFiles,
					baseStyle.Foreground(t.TextMuted()).Render(message),
				),
			)
	}

	// Sort file paths alphabetically for consistent ordering
	var paths []string
	for path := range m.modFiles {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	// Create views for each file in sorted order
	var fileViews []string
	for _, path := range paths {
		stats := m.modFiles[path]
		fileViews = append(fileViews, m.modifiedFile(path, stats.additions, stats.removals))
	}

	return baseStyle.
		Width(m.width).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				modifiedFiles,
				lipgloss.JoinVertical(
					lipgloss.Left,
					fileViews...,
				),
			),
		)
}

func (m *sidebarCmp) SetSize(width, height int) tea.Cmd {
	m.width = width
	m.height = height
	return nil
}

func (m *sidebarCmp) GetSize() (int, int) {
	return m.width, m.height
}

func NewSidebarCmp(session session.Session, history history.Service) tea.Model {
	return &sidebarCmp{
		session: session,
		history: history,
	}
}

// processFileChangesAsync computes one file's diff stats off the Update
// loop (initial-version DB lookup + GenerateDiff) and posts
// fileDiffComputedMsg. Replaces the synchronous processFileChanges/
// findInitialVersion pair (audit H11).
func (m *sidebarCmp) processFileChangesAsync(file history.File) tea.Cmd {
	if m.history == nil {
		return nil
	}
	// Skip if this is the initial version (no changes to show)
	if file.Version == history.InitialVersion {
		return nil
	}
	sid := m.session.ID
	historySvc := m.history
	return func() tea.Msg {
		ctx := context.Background()
		initialVersion, err := func() (history.File, error) {
			fileVersions, err := historySvc.ListBySession(ctx, sid)
			if err != nil {
				return history.File{}, err
			}
			for _, v := range fileVersions {
				if v.Path == file.Path && v.Version == history.InitialVersion {
					return v, nil
				}
			}
			return history.File{}, fmt.Errorf("initial version not found")
		}()
		if err != nil || initialVersion.ID == "" {
			// No initial version to compare against — nothing to show.
			return fileDiffComputedMsg{sessionID: sid, path: file.Path, present: false}
		}

		if initialVersion.Content == file.Content {
			// File reverted to its initial version — drop it from the list.
			return fileDiffComputedMsg{sessionID: sid, path: file.Path, present: false}
		}

		_, additions, removals := diff.GenerateDiff(initialVersion.Content, file.Content, file.Path)
		return fileDiffComputedMsg{
			sessionID: sid,
			path:      file.Path,
			present:   additions > 0 || removals > 0,
			additions: additions,
			removals:  removals,
		}
	}
}

// Helper function to get the display path for a file
func getDisplayPath(path string) string {
	workingDir := config.WorkingDirectory()
	displayPath := strings.TrimPrefix(path, workingDir)
	return strings.TrimPrefix(displayPath, "/")
}
