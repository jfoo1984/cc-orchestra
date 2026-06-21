package tui

import (
	"testing"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

// TestScrollFollowsCursor constructs a model with a small viewport (listHeight ≈ 2)
// and 6 sessions, then moves the cursor down repeatedly.  It asserts that m.top
// advances so that the cursor always remains within [m.top, m.top+listHeight()).
func TestScrollFollowsCursor(t *testing.T) {
	sessions := []session.Session{
		{UUID: "1", Project: "one"},
		{UUID: "2", Project: "two"},
		{UUID: "3", Project: "three"},
		{UUID: "4", Project: "four"},
		{UUID: "5", Project: "five"},
		{UUID: "6", Project: "six"},
	}
	m := loaded(sessions)
	// height=4 → listHeight() = 4-4 = 0, clamped to 1.
	// height=6 → listHeight() = 6-4 = 2.
	m.height = 6

	for step := 0; step < len(sessions)-1; step++ {
		out, _ := m.Update(keyMsg("j"))
		m = out.(Model)

		h := m.listHeight()
		if m.cursor < m.top || m.cursor >= m.top+h {
			t.Fatalf("step %d: cursor %d not in [%d, %d) (listHeight=%d)",
				step, m.cursor, m.top, m.top+h, h)
		}
	}

	// Verify we actually scrolled (top should have advanced from 0).
	if m.top == 0 {
		t.Fatal("expected m.top > 0 after scrolling past the viewport, but it remained 0")
	}
}
