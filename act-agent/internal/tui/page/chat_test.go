package page

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/chat"
)

func TestChatPage_ScrollRouting(t *testing.T) {
	p := NewChatPage(nil)
	cp, ok := p.(*chatPage)
	if !ok {
		t.Fatalf("expected *chatPage, got %T", p)
	}

	// 1. When scrollFocused is false, ctrl+up should route to ScrollMsg command
	cp.scrollFocused = false
	msg := tea.KeyPressMsg(tea.Key{Text: "ctrl+up"})

	_, cmd := cp.Update(msg)
	if cmd == nil {
		t.Fatal("expected ScrollMsg cmd, got nil")
	}

	// Run command and verify it returns chat.ScrollMsg
	resolvedMsg := cmd()
	scrollMsg, ok := resolvedMsg.(chat.ScrollMsg)
	if !ok {
		t.Fatalf("expected chat.ScrollMsg, got %T", resolvedMsg)
	}
	if scrollMsg.Lines != -1 {
		t.Errorf("expected -1 scroll lines, got %d", scrollMsg.Lines)
	}

	// 2. When scrollFocused is true, tab/esc should toggle scrollFocused off, and normal keys are swallowed
	cp.scrollFocused = true
	msgEsc := tea.KeyPressMsg(tea.Key{Text: "esc"})
	_, cmdEsc := cp.Update(msgEsc)
	if cmdEsc == nil {
		t.Fatal("expected ScrollFocusMsg cmd, got nil")
	}
	focusMsg := cmdEsc()
	focusMsgTyped, ok := focusMsg.(chat.ScrollFocusMsg)
	if !ok || focusMsgTyped.On {
		t.Errorf("expected ScrollFocusMsg{On: false}, got %v", focusMsg)
	}
}
