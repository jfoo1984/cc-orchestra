package tui

import (
	"testing"
	"time"
)

func TestPollRearmsTicker(t *testing.T) {
	m := loaded(nil)
	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("tickMsg should return a batch (refresh + re-arm)")
	}
}
