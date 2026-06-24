package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissing(t *testing.T) {
	r, err := Load(filepath.Join(t.TempDir(), "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != Version || r.Sessions == nil || len(r.Sessions) != 0 {
		t.Fatalf("missing load not empty: %+v", r)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, _ := Load(path)
	if err := r.Update("u1", func(e *Entry) { e.Name = "alpha"; e.Pinned = true }); err != nil {
		t.Fatal(err)
	}
	got, _ := Load(path)
	if e := got.Sessions["u1"]; e.Name != "alpha" || !e.Pinned || e.UpdatedAt.IsZero() {
		t.Fatalf("round trip wrong: %+v", e)
	}
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "*.tmp"))
	if len(matches) != 0 {
		t.Fatalf("temp file left behind: %v", matches)
	}
}

func TestLoadCorrupt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Load(path)
	if err != nil || len(r.Sessions) != 0 {
		t.Fatalf("corrupt load: got (%+v,%v)", r, err)
	}
	backups, _ := filepath.Glob(path + ".corrupt-*")
	if len(backups) != 1 {
		t.Fatalf("want 1 corrupt backup, got %v", backups)
	}
}

func TestMetas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, _ := Load(path)
	_ = r.Update("u1", func(e *Entry) { e.Name = "x"; e.Archived = true })
	m := r.Metas()
	if m["u1"].DisplayName != "x" || !m["u1"].Archived {
		t.Fatalf("Metas wrong: %+v", m["u1"])
	}
}
