package sources

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTranscript(t *testing.T, root, project, uuid string, lines []string) {
	t.Helper()
	dir := filepath.Join(root, project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := ""
	for _, l := range lines {
		body += l + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, uuid+".jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	// Normal session: a mode line (no cwd), a tool-result user line, then the real prompt.
	writeTranscript(t, root, "-home-me-alpha", "u-normal", []string{
		`{"type":"mode","mode":"default","sessionId":"u-normal"}`,
		`not even json`,
		`{"type":"user","isMeta":false,"cwd":"/home/me/alpha","gitBranch":"main","toolUseResult":{"x":1},"message":{"role":"user","content":[{"type":"tool_result"}]}}`,
		`{"type":"user","isMeta":false,"cwd":"/home/me/alpha","gitBranch":"main","message":{"role":"user","content":"fix the parser"}}`,
		`{"type":"assistant","cwd":"/home/me/alpha","message":{"role":"assistant","content":"ok"}}`,
	})
	// Sidechain: must be skipped entirely.
	writeTranscript(t, root, "-home-me-beta", "u-side", []string{
		`{"type":"user","isSidechain":true,"cwd":"/home/me/beta","message":{"role":"user","content":"sub"}}`,
	})

	got, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 transcript, got %d (%+v)", len(got), got)
	}
	ti := got[0]
	if ti.UUID != "u-normal" || ti.Cwd != "/home/me/alpha" || ti.GitBranch != "main" {
		t.Errorf("metadata wrong: %+v", ti)
	}
	if ti.FirstUserMsg != "fix the parser" {
		t.Errorf("FirstUserMsg = %q, want 'fix the parser'", ti.FirstUserMsg)
	}
	if ti.LastActive.IsZero() {
		t.Errorf("LastActive should be the file mtime, got zero")
	}
}

func TestScanMissingRoot(t *testing.T) {
	got, err := Scan(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil || got != nil {
		t.Fatalf("missing root: got (%v, %v), want (nil, nil)", got, err)
	}
}
