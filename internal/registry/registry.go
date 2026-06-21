package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

const Version = 1

type Entry struct {
	Name      string    `json:"name,omitempty"`
	Pinned    bool      `json:"pinned,omitempty"`
	Archived  bool      `json:"archived,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type Registry struct {
	Version  int              `json:"version"`
	Sessions map[string]Entry `json:"sessions"`
	path     string
}

// DefaultPath honors $XDG_STATE_HOME, else ~/.local/state.
func DefaultPath() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "cc-orchestra", "registry.json"), nil
}

// Load reads the registry. Missing file → empty registry. Corrupt file → backed
// up to *.corrupt-<unix> and treated as empty.
func Load(path string) (*Registry, error) {
	empty := &Registry{Version: Version, Sessions: map[string]Entry{}, path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return empty, nil
		}
		return nil, err
	}
	var parsed Registry
	if err := json.Unmarshal(data, &parsed); err != nil {
		_ = os.Rename(path, fmt.Sprintf("%s.corrupt-%d", path, time.Now().Unix()))
		return empty, nil
	}
	if parsed.Sessions == nil {
		parsed.Sessions = map[string]Entry{}
	}
	parsed.Version = Version
	parsed.path = path
	return &parsed, nil
}

// Save writes the registry atomically (temp file + fsync + rename).
func (r *Registry) Save() error {
	if r.path == "" {
		return fmt.Errorf("registry: no path set")
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(r.path), "registry-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // best-effort; no-op once renamed
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, r.path)
}

// Metas converts entries to session.Meta keyed by UUID.
func (r *Registry) Metas() map[string]session.Meta {
	m := make(map[string]session.Meta, len(r.Sessions))
	for uuid, e := range r.Sessions {
		m[uuid] = session.Meta{DisplayName: e.Name, Pinned: e.Pinned, Archived: e.Archived, Notes: e.Notes}
	}
	return m
}

// Update mutates the entry for uuid, stamps UpdatedAt, and saves.
func (r *Registry) Update(uuid string, fn func(*Entry)) error {
	if r.Sessions == nil {
		r.Sessions = map[string]Entry{}
	}
	e := r.Sessions[uuid]
	fn(&e)
	e.UpdatedAt = time.Now().UTC()
	r.Sessions[uuid] = e
	return r.Save()
}
