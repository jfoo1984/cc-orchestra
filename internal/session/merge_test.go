package session

import (
	"testing"
	"time"
)

func TestMerge(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	transcripts := []TranscriptInfo{
		{UUID: "u-fresh", Cwd: "/home/me/alpha", LastActive: now.Add(-2 * time.Second), Path: "/t/u-fresh.jsonl"},
		{UUID: "u-stale", Cwd: "/home/me/beta", LastActive: now.Add(-time.Hour), Path: "/t/u-stale.jsonl"},
		{UUID: "u-cold", Cwd: "/home/me/gamma", LastActive: now.Add(-48 * time.Hour), Path: "/t/u-cold.jsonl"},
	}
	live := []LiveInfo{
		{UUID: "u-fresh", PID: 10, Cwd: "/home/me/alpha", StartedAt: now.Add(-time.Minute)},
		{UUID: "u-stale", PID: 11, Cwd: "/home/me/beta", StartedAt: now.Add(-time.Hour)},
		{UUID: "u-liveonly", PID: 12, Cwd: "/home/me/delta", StartedAt: now.Add(-3 * time.Second)},
	}
	meta := map[string]Meta{
		"u-cold":   {Pinned: true, DisplayName: "pinned cold"},
		"u-orphan": {DisplayName: "ghost"},
	}

	got := Merge(transcripts, live, meta, now)

	byUUID := map[string]Session{}
	for _, s := range got {
		byUUID[s.UUID] = s
	}
	if _, ok := byUUID["u-orphan"]; ok {
		t.Fatalf("registry-only orphan should not be surfaced")
	}
	if byUUID["u-fresh"].Status != StatusBusy {
		t.Errorf("u-fresh: want busy, got %v", byUUID["u-fresh"].Status)
	}
	if byUUID["u-stale"].Status != StatusIdle {
		t.Errorf("u-stale: want idle, got %v", byUUID["u-stale"].Status)
	}
	if byUUID["u-cold"].Status != StatusNotRunning {
		t.Errorf("u-cold: want not-running, got %v", byUUID["u-cold"].Status)
	}
	if s := byUUID["u-liveonly"]; s.Cwd != "/home/me/delta" || s.Project != "delta" || s.Status != StatusBusy {
		t.Errorf("u-liveonly merge wrong: %+v", s)
	}
	if !byUUID["u-cold"].Pinned || byUUID["u-cold"].DisplayName != "pinned cold" {
		t.Errorf("meta not applied to u-cold")
	}
	// Sort: pinned u-cold first, then busy sessions, then idle.
	if got[0].UUID != "u-cold" {
		t.Errorf("pinned session should sort first, got %q", got[0].UUID)
	}
}
