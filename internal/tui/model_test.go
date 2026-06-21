package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

func fixedNow() time.Time { return time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC) }

func keyMsg(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func loaded(sessions []session.Session) Model {
	m := New(func() ([]session.Session, error) { return sessions, nil }, nil, nil, fixedNow)
	m.height = 24
	out, _ := m.Update(fleetMsg{sessions: sessions})
	return out.(Model)
}

func TestNavAndArchived(t *testing.T) {
	sessions := []session.Session{
		{UUID: "a", Project: "alpha"},
		{UUID: "b", Project: "beta", Archived: true},
		{UUID: "c", Project: "gamma"},
	}
	m := loaded(sessions)
	if len(m.visible) != 2 {
		t.Fatalf("archived hidden: want 2 visible, got %d", len(m.visible))
	}
	out, _ := m.Update(keyMsg("j"))
	m = out.(Model)
	if m.cursor != 1 {
		t.Fatalf("cursor after j = %d, want 1", m.cursor)
	}
	out, _ = m.Update(keyMsg("A"))
	m = out.(Model)
	if len(m.visible) != 3 {
		t.Fatalf("show archived: want 3 visible, got %d", len(m.visible))
	}
}

func TestQuit(t *testing.T) {
	m := loaded(nil)
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q should return a command (tea.Quit)")
	}
}
