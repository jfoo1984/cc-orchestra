package session

import (
	"strings"
	"testing"
	"time"
)

func TestNameResolution(t *testing.T) {
	tests := []struct {
		name string
		s    Session
		want string
	}{
		{"display name wins", Session{DisplayName: "deploy bot", FirstUserMsg: "hi", Project: "p", UUID: "abcdef12-0000"}, "deploy bot"},
		{"first message next", Session{FirstUserMsg: "  fix the\nflaky test  ", Project: "p", UUID: "abcdef12-0000"}, "fix the flaky test"},
		{"project next", Session{Project: "myproj", UUID: "abcdef12-0000"}, "myproj"},
		{"short uuid fallback", Session{UUID: "abcdef12-0000"}, "abcdef12"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.Name(); got != tc.want {
				t.Fatalf("Name() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGlyphAndStatusText(t *testing.T) {
	cases := map[Status][2]string{
		StatusBusy:       {"●", "busy"},
		StatusIdle:       {"◐", "idle"},
		StatusNotRunning: {"○", "—"},
	}
	for st, want := range cases {
		s := Session{Status: st}
		if s.Glyph() != want[0] || s.StatusText() != want[1] {
			t.Fatalf("status %d: got (%s,%s) want (%s,%s)", st, s.Glyph(), s.StatusText(), want[0], want[1])
		}
	}
}

func TestAge(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		ago  time.Duration
		want string
	}{
		{0, "now"},
		{30 * time.Second, "now"},
		{5 * time.Minute, "5m"},
		{3 * time.Hour, "3h"},
		{50 * time.Hour, "2d"},
	}
	for _, c := range cases {
		if got := Age(now.Add(-c.ago), now); got != c.want {
			t.Fatalf("Age(-%s) = %q, want %q", c.ago, got, c.want)
		}
	}
	if got := Age(time.Time{}, now); got != "—" {
		t.Fatalf("Age(zero) = %q, want —", got)
	}
}

func TestSanitizeTruncate(t *testing.T) {
	long := strings.Repeat("a", 80)
	got := truncate(sanitize("x\n\t y"), 60)
	if got != "x y" {
		t.Fatalf("sanitize/truncate = %q", got)
	}
	if r := []rune(truncate(long, 60)); len(r) != 60 || r[59] != '…' {
		t.Fatalf("truncate did not cap at 60 runes with ellipsis")
	}
}
