package session

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Status is derived, not read from the CLI: "running" means the UUID appears in
// `claude agents --json`; busy-vs-idle is inferred from transcript mtime freshness.
type Status int

const (
	StatusNotRunning Status = iota // not present in `claude agents --json`
	StatusIdle                     // running, transcript mtime older than BusyThreshold
	StatusBusy                     // running, transcript modified within BusyThreshold
)

// BusyThreshold separates "busy" from "idle" for running sessions.
const BusyThreshold = 10 * time.Second

type TokenUsage struct {
	Input  int
	Output int
	Total  int
}

// Detail holds lazily-loaded preview fields for a session.
type Detail struct {
	Model       string
	Tokens      TokenUsage
	LastUserMsg string
	LastAsstMsg string
}

type Session struct {
	UUID         string
	Cwd          string
	Project      string
	GitBranch    string
	FirstUserMsg string

	Status     Status
	PID        int
	StartedAt  time.Time
	LastActive time.Time

	// Registry-derived metadata:
	DisplayName string
	Pinned      bool
	Archived    bool
	Notes       string

	// Provenance:
	HasTranscript  bool
	IsLive         bool
	TranscriptPath string
}

// Name resolves the display name: registry name → first user message → project → short UUID.
func (s Session) Name() string {
	switch {
	case s.DisplayName != "":
		return s.DisplayName
	case s.FirstUserMsg != "":
		return truncate(sanitize(s.FirstUserMsg), 60)
	case s.Project != "":
		return s.Project
	case len(s.UUID) >= 8:
		return s.UUID[:8]
	default:
		return s.UUID
	}
}

func (s Session) Glyph() string {
	switch s.Status {
	case StatusBusy:
		return "●"
	case StatusIdle:
		return "◐"
	default:
		return "○"
	}
}

func (s Session) StatusText() string {
	switch s.Status {
	case StatusBusy:
		return "busy"
	case StatusIdle:
		return "idle"
	default:
		return "—"
	}
}

// Age renders a short relative duration like "now", "5m", "2h", "3d".
func Age(t, now time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// Basename returns the last path element of a cwd, used for Project.
func Basename(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

func sanitize(s string) string { return strings.Join(strings.Fields(s), " ") }

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
