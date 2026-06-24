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

// TestStaleDetailRejected verifies that a detailMsg for a non-selected session
// does not overwrite m.preview or m.previewUUID.
func TestStaleDetailRejected(t *testing.T) {
	sessions := []session.Session{
		{UUID: "a", Project: "alpha"},
		{UUID: "b", Project: "beta"},
	}
	m := loaded(sessions) // cursor=0 → "a" is selected

	// First, set a preview for "a" so we have something to protect.
	out, _ := m.Update(detailMsg{uuid: "a", detail: session.Detail{Model: "claude-3"}})
	m = out.(Model)
	savedPreview := m.preview
	savedUUID := m.previewUUID
	if savedPreview == "" {
		t.Fatal("precondition: preview for 'a' should have been set")
	}

	// Now deliver a stale detailMsg for "b" while "a" is still selected.
	out, _ = m.Update(detailMsg{uuid: "b", detail: session.Detail{Model: "claude-stale"}})
	m = out.(Model)
	if m.preview != savedPreview || m.previewUUID != savedUUID {
		t.Fatalf("stale detailMsg for 'b' mutated preview: preview=%q uuid=%q", m.preview, m.previewUUID)
	}
}

// TestStalePreviewTickRejected verifies that a previewTickMsg for a non-selected
// uuid returns nil cmd (no load is fired).
func TestStalePreviewTickRejected(t *testing.T) {
	sessions := []session.Session{
		{UUID: "a", Project: "alpha"},
		{UUID: "b", Project: "beta"},
	}
	m := loaded(sessions) // cursor=0 → "a" is selected

	// Deliver a tick for "b" (not selected); expect nil cmd.
	_, cmd := m.Update(previewTickMsg{uuid: "b"})
	if cmd != nil {
		t.Fatal("stale previewTickMsg for non-selected uuid should return nil cmd, got non-nil")
	}
}
