package tui

import (
	"testing"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

func TestPreviewRendersDetail(t *testing.T) {
	s := session.Session{UUID: "a", Project: "alpha", Cwd: "/p/alpha", Status: session.StatusIdle}
	m := loaded([]session.Session{s})
	out, _ := m.Update(detailMsg{uuid: "a", detail: session.Detail{Model: "claude-opus-4-8"}})
	m = out.(Model)
	if m.preview == "" || m.previewUUID != "a" {
		t.Fatalf("preview not set: %q / %q", m.preview, m.previewUUID)
	}
}
