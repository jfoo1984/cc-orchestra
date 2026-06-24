package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

func TestFilterNarrowsAndClears(t *testing.T) {
	m := loaded(sessionsFixture())
	out, _ := m.Update(keyMsg("/"))
	m = out.(Model)
	if !m.filtering {
		t.Fatal("/ should enter filter mode")
	}
	for _, r := range "alp" {
		out, _ = m.Update(keyMsg(string(r)))
		m = out.(Model)
	}
	if len(m.visible) != 1 || m.visible[0].Project != "alpha" {
		t.Fatalf("filter 'alp' should match only alpha, got %d", len(m.visible))
	}
	out, _ = m.Update(tea_esc())
	m = out.(Model)
	if m.filtering || len(m.visible) != 3 {
		t.Fatalf("esc should clear filter; visible=%d filtering=%v", len(m.visible), m.filtering)
	}
}

// TestFilterAndArchivedIntersection verifies that an archived session matching the
// active filter is hidden while showArchived is false, and becomes visible only after
// toggling showArchived on (both conditions must be satisfied simultaneously).
func TestFilterAndArchivedIntersection(t *testing.T) {
	sessions := []session.Session{
		{UUID: "a", Project: "alpha"},
		{UUID: "b", Project: "beta-archive", Archived: true},
		{UUID: "c", Project: "gamma"},
	}
	m := loaded(sessions)

	// Enter filter mode and type "beta" — matches the archived session.
	out, _ := m.Update(keyMsg("/"))
	m = out.(Model)
	for _, r := range "beta" {
		out, _ = m.Update(keyMsg(string(r)))
		m = out.(Model)
	}
	// Archived row must still be hidden even though it matches the filter.
	for _, s := range m.visible {
		if s.UUID == "b" {
			t.Fatal("archived session 'b' must not appear while showArchived=false, even when filter matches")
		}
	}

	// Exit filter mode (keep the filter value active via enter, not esc).
	out, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = out.(Model)
	if m.filtering {
		t.Fatal("enter should exit filter mode")
	}

	// Now toggle showArchived on via "A".
	out, _ = m.Update(keyMsg("A"))
	m = out.(Model)

	found := false
	for _, s := range m.visible {
		if s.UUID == "b" {
			found = true
		}
	}
	if !found {
		t.Fatal("archived session 'b' should appear after showArchived toggled on with matching filter")
	}
}
