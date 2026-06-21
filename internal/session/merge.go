package session

import (
	"sort"
	"time"
)

type TranscriptInfo struct {
	UUID         string
	Cwd          string
	GitBranch    string
	FirstUserMsg string
	LastActive   time.Time
	Path         string
}

type LiveInfo struct {
	UUID      string
	PID       int
	Cwd       string
	StartedAt time.Time
}

type Meta struct {
	DisplayName string
	Pinned      bool
	Archived    bool
	Notes       string
}

// Merge unifies transcripts, live agents, and registry metadata into a sorted
// []Session keyed by UUID. Registry-only entries (no transcript, no live) are
// not surfaced. `now` is injected for deterministic tests.
func Merge(transcripts []TranscriptInfo, live []LiveInfo, meta map[string]Meta, now time.Time) []Session {
	byUUID := make(map[string]*Session, len(transcripts)+len(live))

	for _, t := range transcripts {
		byUUID[t.UUID] = &Session{
			UUID:           t.UUID,
			Cwd:            t.Cwd,
			Project:        Basename(t.Cwd),
			GitBranch:      t.GitBranch,
			FirstUserMsg:   t.FirstUserMsg,
			LastActive:     t.LastActive,
			Status:         StatusNotRunning,
			HasTranscript:  true,
			TranscriptPath: t.Path,
		}
	}

	for _, l := range live {
		s := byUUID[l.UUID]
		if s == nil {
			s = &Session{UUID: l.UUID}
			byUUID[l.UUID] = s
		}
		s.IsLive = true
		s.PID = l.PID
		s.StartedAt = l.StartedAt
		if s.Cwd == "" {
			s.Cwd = l.Cwd
			s.Project = Basename(l.Cwd)
		}
		ref := s.LastActive
		if ref.IsZero() {
			ref = l.StartedAt
		}
		if now.Sub(ref) < BusyThreshold {
			s.Status = StatusBusy
		} else {
			s.Status = StatusIdle
		}
	}

	out := make([]Session, 0, len(byUUID))
	for uuid, s := range byUUID {
		if m, ok := meta[uuid]; ok {
			s.DisplayName = m.DisplayName
			s.Pinned = m.Pinned
			s.Archived = m.Archived
			s.Notes = m.Notes
		}
		out = append(out, *s)
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Pinned != b.Pinned {
			return a.Pinned
		}
		if a.Status != b.Status {
			return a.Status > b.Status
		}
		return a.LastActive.After(b.LastActive)
	})
	return out
}
