package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/app"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/history"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/permission"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
)

type mockSessionService struct {
	session.Service
}

func (m *mockSessionService) Subscribe(ctx context.Context) <-chan pubsub.Event[session.Session] {
	return make(chan pubsub.Event[session.Session])
}

func (m *mockSessionService) List(ctx context.Context) ([]session.Session, error) {
	return nil, nil
}

func (m *mockSessionService) Save(ctx context.Context, s session.Session) (session.Session, error) {
	return s, nil
}

type mockMessageService struct {
	message.Service
}

func (m *mockMessageService) Subscribe(ctx context.Context) <-chan pubsub.Event[message.Message] {
	return make(chan pubsub.Event[message.Message])
}

type mockHistoryService struct {
	history.Service
}

func (m *mockHistoryService) Subscribe(ctx context.Context) <-chan pubsub.Event[history.File] {
	return make(chan pubsub.Event[history.File])
}

type mockPermissionService struct {
	permission.Service
}

func (m *mockPermissionService) Subscribe(ctx context.Context) <-chan pubsub.Event[permission.PermissionRequest] {
	return make(chan pubsub.Event[permission.PermissionRequest])
}

func TestAppModel_RenamePrecedence(t *testing.T) {
	a := &app.App{
		Sessions:    &mockSessionService{},
		Messages:    &mockMessageService{},
		History:     &mockHistoryService{},
		Permissions: &mockPermissionService{},
	}

	// Create a new appModel
	model := New(a)
	am, ok := model.(*appModel)
	if !ok {
		t.Fatalf("expected *appModel, got %T", model)
	}

	// 1. Initially, the rename modal is not on the stack
	if am.isModalActive(ModalRename) {
		t.Fatal("expected ModalRename to be inactive initially")
	}

	// 2. Select a session so renaming is allowed
	am.selectedSession = session.Session{ID: "session-123", Title: "Original Title"}

	// 3. Send RenameSession keypress (ctrl+e)
	msg := tea.KeyPressMsg(tea.Key{Text: "ctrl+e"})
	updatedModel, _ := am.Update(msg)
	updatedAm, ok := updatedModel.(appModel)
	if !ok {
		t.Fatalf("expected appModel, got %T", updatedModel)
	}

	// Verification — the rename modal must now be the TOP of the stack
	if !updatedAm.isModalActive(ModalRename) {
		t.Error("expected ModalRename active after ctrl+e keypress")
	}
	if top, ok := updatedAm.topModal(); !ok || top != ModalRename {
		t.Errorf("expected ModalRename as stack top, got %v (ok=%v)", top, ok)
	}
}
