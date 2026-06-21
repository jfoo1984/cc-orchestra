package tui

import "testing"

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
