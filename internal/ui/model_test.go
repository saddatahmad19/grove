package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saddatahmad19/grove/internal/config"
)

func TestNewModelDoesNotPanicWithoutRepo(t *testing.T) {
	m := NewModel(config.Config{Root: t.TempDir()})
	if m.View() == "" {
		t.Fatal("expected view to render")
	}
}

func TestCreatePromptFlow(t *testing.T) {
	m := NewModel(config.Config{Root: t.TempDir()})
	m.viewState = stateRepoLoaded
	m.mode = modeNavigate

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	um := updated.(Model)
	if um.mode != modeCreatePrompt {
		t.Fatalf("expected create prompt mode, got %v", um.mode)
	}

	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	um = updated.(Model)
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	um = updated.(Model)
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um = updated.(Model)
	if um.mode != modeNavigate {
		t.Fatalf("expected navigate mode after confirm, got %v", um.mode)
	}
}
