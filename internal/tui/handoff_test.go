package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

type fakeHandoff struct{ got string }

func (f *fakeHandoff) Run(uuid string) tea.Cmd {
	f.got = uuid
	return nil
}

func TestHandoffUsesSelectedUUID(t *testing.T) {
	fh := &fakeHandoff{}
	sessions := []session.Session{{UUID: "u-1", Project: "alpha"}, {UUID: "u-2", Project: "beta"}}
	m := New(func() ([]session.Session, error) { return sessions, nil }, nil, fh, func() time.Time { return fixedNow() })
	m.height = 24
	out, _ := m.Update(fleetMsg{sessions: sessions})
	m = out.(Model)

	out, _ = m.Update(keyMsg("j")) // move to u-2
	m = out.(Model)
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		cmd() // fakeHandoff.Run returns nil, but doHandoff invoked it
	}
	if fh.got != "u-2" {
		t.Fatalf("handoff got %q, want u-2", fh.got)
	}
}
