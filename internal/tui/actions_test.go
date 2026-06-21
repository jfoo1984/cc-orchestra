package tui

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jfoo1984/cc-orchestra/internal/registry"
	"github.com/jfoo1984/cc-orchestra/internal/session"
)

func modelWithReg(t *testing.T, sessions []session.Session) (Model, *registry.Registry) {
	t.Helper()
	reg, _ := registry.Load(filepath.Join(t.TempDir(), "registry.json"))
	loader := func() ([]session.Session, error) {
		// Re-apply registry metadata so refreshes reflect mutations.
		metas := reg.Metas()
		out := make([]session.Session, 0, len(sessions))
		for _, s := range sessions {
			if mm, ok := metas[s.UUID]; ok {
				s.Pinned, s.Archived, s.DisplayName = mm.Pinned, mm.Archived, mm.DisplayName
			}
			out = append(out, s)
		}
		return out, nil
	}
	m := New(loader, reg, nil, func() time.Time { return fixedNow() })
	m.height = 24
	out, _ := m.Update(fleetMsg{sessions: sessions})
	return out.(Model), reg
}

func TestActionsPinAndRename(t *testing.T) {
	m, reg := modelWithReg(t, []session.Session{{UUID: "a", Project: "alpha"}})

	out, cmd := m.Update(keyMsg("p"))
	m = out.(Model)
	if cmd == nil {
		t.Fatal("pin should trigger a refresh command")
	}
	if !reg.Sessions["a"].Pinned {
		t.Fatal("pin not persisted to registry")
	}

	out, _ = m.Update(keyMsg("n"))
	m = out.(Model)
	for _, r := range "bot" {
		out, _ = m.Update(keyMsg(string(r)))
		m = out.(Model)
	}
	out, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = out.(Model)
	if reg.Sessions["a"].Name != "bot" {
		t.Fatalf("rename not persisted, got %q", reg.Sessions["a"].Name)
	}
}
