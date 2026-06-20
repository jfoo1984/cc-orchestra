# cc-orchestra Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-extended-cc:subagent-driven-development (recommended) or superpowers-extended-cc:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build cc-orchestra, a single-binary Go/Bubble Tea TUI that lists, previews, and hands off into local Claude Code sessions.

**Architecture:** Three data sources — on-disk transcripts (`~/.claude/projects/**/<uuid>.jsonl`), `claude agents --json`, and a JSON registry of our own metadata — are merged into a UUID-keyed `Session` model. A Bubble Tea TUI renders the fleet and uses `tea.ExecProcess` to suspend itself, run `claude --resume <uuid>`, and resume — so the TUI is a persistent home base.

**Tech Stack:** Go 1.24; Bubble Tea (`bubbletea`, `bubbles/textinput`, `bubbles/key`, `lipgloss`); standard library for JSON / file IO / `os/exec`.

**User decisions (already made):**
- "cc-orchestra everywhere" — module `github.com/jfoo1984/cc-orchestra`, binary `cc-orchestra`, registry `~/.local/state/cc-orchestra/`.
- Public repo; day-one CI (`go test` + golangci-lint); `go install` distribution; MIT license (already committed).
- Hand-off = "TUI as home base" via `tea.ExecProcess` (not `syscall.Exec`).
- Status = "ternary (mtime heuristic)": ● busy (running + transcript mtime within `BusyThreshold`) · ◐ idle (running, stale) · ○ not-running. "Running" = UUID present in `claude agents --json`.
- Spec: `docs/superpowers/specs/2026-06-20-cc-orchestra-design.md`.

**Sequencing note:** Tasks 0–6 (data layer) are independent and testable in isolation. Tasks 7–13 (TUI + wiring) build sequentially on the `tui.Model` introduced in Task 7; implement them in order.

---

### Task 0: Project scaffold

**Goal:** A building, vetted Go module skeleton with CI and lint config.

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `.golangci.yml`
- Create: `.github/workflows/ci.yml`
- Create: `README.md`
- Create: `cmd/cc-orchestra/main.go`

**Acceptance Criteria:**
- [ ] `go build ./...` succeeds.
- [ ] `go vet ./...` is clean.
- [ ] Running the binary prints `cc-orchestra <version>`.

**Verify:** `go build ./... && go vet ./...` → exits 0, no output.

**Steps:**

- [ ] **Step 1: Initialize the module**

```bash
go mod init github.com/jfoo1984/cc-orchestra
```

- [ ] **Step 2: Create `cmd/cc-orchestra/main.go`** (dependency-free stub; real wiring lands in Task 13)

```go
package main

import "fmt"

// version is overridden at release time via -ldflags.
var version = "dev"

func main() {
	fmt.Printf("cc-orchestra %s\n", version)
}
```

- [ ] **Step 3: Create `Makefile`**

```makefile
BINARY := cc-orchestra

.PHONY: build test lint install run tidy
build:
	go build -o bin/$(BINARY) ./cmd/cc-orchestra
test:
	go test ./...
lint:
	golangci-lint run
install:
	go install ./cmd/cc-orchestra
run: build
	./bin/$(BINARY)
tidy:
	go mod tidy
```

- [ ] **Step 4: Create `.golangci.yml`**

```yaml
run:
  timeout: 5m
linters:
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell
    - revive
```

- [ ] **Step 5: Create `.github/workflows/ci.yml`**

```yaml
name: CI
on:
  push:
    branches: [ main ]
  pull_request:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go test ./...
      - run: go vet ./...
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - uses: golangci/golangci-lint-action@v6
        with:
          version: v1.61.0
```

- [ ] **Step 6: Create `README.md`** (skeleton; finalized in Task 13)

```markdown
# cc-orchestra

A terminal UI for managing your local Claude Code sessions: browse every session
on disk as a fleet, see which are running, and jump into one.

## Install

    go install github.com/jfoo1984/cc-orchestra/cmd/cc-orchestra@latest

## Status

Under construction. See `docs/superpowers/specs/2026-06-20-cc-orchestra-design.md`.
```

- [ ] **Step 7: Verify and commit**

```bash
go build ./... && go vet ./...
git add go.mod Makefile .golangci.yml .github README.md cmd
git commit -m "Scaffold cc-orchestra module, CI, and lint config"
```

---

### Task 1: Session core types and formatting

**Goal:** The `session.Session` value type plus the derived display helpers (name resolution, status glyph/text, relative age).

**Files:**
- Create: `internal/session/session.go`
- Test: `internal/session/session_test.go`

**Acceptance Criteria:**
- [ ] `Status` is `StatusNotRunning=0 < StatusIdle < StatusBusy` (so higher = more active).
- [ ] `Name()` resolves `DisplayName` → sanitized+truncated `FirstUserMsg` → `Project` → short UUID.
- [ ] `Glyph()` returns ● / ◐ / ○ and `StatusText()` returns busy / idle / —.
- [ ] `Age()` renders now / `%dm` / `%dh` / `%dd` and `—` for the zero time.

**Verify:** `go test ./internal/session/ -run 'TestName|TestGlyph|TestStatusText|TestAge|TestSanitize' -v` → PASS.

**Steps:**

- [ ] **Step 1: Write `internal/session/session_test.go`**

```go
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
```

- [ ] **Step 2: Run the test, expect FAIL** — `go test ./internal/session/` → build errors (undefined symbols).

- [ ] **Step 3: Write `internal/session/session.go`**

```go
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

	// Lazy preview fields (loaded on selection):
	Model       string
	Tokens      TokenUsage
	LastUserMsg string
	LastAsstMsg string

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
```

- [ ] **Step 4: Run the test, expect PASS** — `go test ./internal/session/ -v`.

- [ ] **Step 5: Commit**

```bash
git add internal/session/session.go internal/session/session_test.go
git commit -m "Add session core types and display helpers"
```

---

### Task 2: Merge and sort

**Goal:** `session.Merge` unifies transcript-derived sessions, live agents, and registry metadata into a sorted slice, deriving busy/idle from mtime freshness.

**Files:**
- Create: `internal/session/merge.go`
- Test: `internal/session/merge_test.go`

**Acceptance Criteria:**
- [ ] Transcript-only sessions are `StatusNotRunning` with `Project = basename(cwd)`.
- [ ] A running session with transcript mtime within `BusyThreshold` is `StatusBusy`; stale is `StatusIdle`.
- [ ] A live-only session (no transcript) takes `Cwd` from the agent and is included.
- [ ] Registry `Meta` applies pinned/archived/name/notes.
- [ ] Sort order is pinned → busy → idle → not-running, then `LastActive` descending.
- [ ] Registry-only entries (no transcript, no live) are NOT surfaced as rows.

**Verify:** `go test ./internal/session/ -run TestMerge -v` → PASS.

**Steps:**

- [ ] **Step 1: Write `internal/session/merge_test.go`**

```go
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
		"u-cold": {Pinned: true, DisplayName: "pinned cold"},
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
```

- [ ] **Step 2: Run, expect FAIL** (undefined `Merge`, `TranscriptInfo`, `LiveInfo`, `Meta`).

- [ ] **Step 3: Write `internal/session/merge.go`**

```go
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
```

- [ ] **Step 4: Run, expect PASS** — `go test ./internal/session/ -v`.

- [ ] **Step 5: Commit**

```bash
git add internal/session/merge.go internal/session/merge_test.go
git commit -m "Add session merge and sort logic"
```

---

### Task 3: Source paths

**Goal:** Locate the transcripts root and provide a best-effort project-dir decoder (fallback only).

**Files:**
- Create: `internal/sources/paths.go`
- Test: `internal/sources/paths_test.go`

**Acceptance Criteria:**
- [ ] `ProjectsRoot()` honors `$CLAUDE_CONFIG_DIR`, else `~/.claude/projects`.
- [ ] `DecodeProjectDir("-home-user-foo")` returns `/home/user/foo` (documented as lossy).

**Verify:** `go test ./internal/sources/ -run TestPaths -v` → PASS.

**Steps:**

- [ ] **Step 1: Write `internal/sources/paths_test.go`**

```go
package sources

import (
	"path/filepath"
	"testing"
)

func TestPathsProjectsRootEnv(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/custom/cfg")
	got, err := ProjectsRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join("/custom/cfg", "projects") {
		t.Fatalf("ProjectsRoot = %q", got)
	}
}

func TestPathsDecodeProjectDir(t *testing.T) {
	cases := map[string]string{
		"-home-user-foo": "/home/user/foo",
		"":               "",
	}
	for in, want := range cases {
		if got := DecodeProjectDir(in); got != want {
			t.Fatalf("DecodeProjectDir(%q) = %q, want %q", in, got, want)
		}
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Write `internal/sources/paths.go`**

```go
package sources

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectsRoot returns the directory holding per-project transcript folders.
// Honors CLAUDE_CONFIG_DIR; otherwise ~/.claude/projects.
func ProjectsRoot() (string, error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "projects"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// DecodeProjectDir reconstructs a cwd from an encoded project-dir name, e.g.
// "-home-user-foo" → "/home/user/foo". Lossy because path segments may contain
// '-'; prefer the cwd embedded in the transcript. Fallback only.
func DecodeProjectDir(name string) string {
	if name == "" {
		return ""
	}
	return "/" + strings.ReplaceAll(strings.TrimPrefix(name, "-"), "-", "/")
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/sources/paths.go internal/sources/paths_test.go
git commit -m "Add source path helpers"
```

---

### Task 4: Transcript scanning

**Goal:** Scan the projects root and extract one `TranscriptInfo` per top-level (non-sidechain) transcript: cwd, gitBranch, first real user message, and mtime.

**Files:**
- Create: `internal/sources/transcripts.go`
- Test: `internal/sources/transcripts_test.go`

**Acceptance Criteria:**
- [ ] Returns `(nil, nil)` when the root does not exist.
- [ ] Skips `isSidechain:true` transcripts and non-`.jsonl` files.
- [ ] First user message skips `mode`/meta lines and `user` lines carrying `toolUseResult`; uses the first string `message.content`.
- [ ] `cwd`/`gitBranch` come from the transcript body; mtime populates `LastActive`; UUID from the filename.
- [ ] Malformed JSON lines are skipped without error.

**Verify:** `go test ./internal/sources/ -run TestScan -v` → PASS.

**Steps:**

- [ ] **Step 1: Write `internal/sources/transcripts_test.go`**

```go
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
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Write `internal/sources/transcripts.go`**

```go
package sources

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

type rawLine struct {
	Type          string          `json:"type"`
	Cwd           string          `json:"cwd"`
	GitBranch     string          `json:"gitBranch"`
	IsSidechain   bool            `json:"isSidechain"`
	IsMeta        bool            `json:"isMeta"`
	Message       json.RawMessage `json:"message"`
	ToolUseResult json.RawMessage `json:"toolUseResult"`
}

type rawMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// Scan walks the projects root and returns one TranscriptInfo per top-level
// (non-sidechain) session transcript. A missing root yields (nil, nil).
func Scan(root string) ([]session.TranscriptInfo, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []session.TranscriptInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			name := f.Name()
			if f.IsDir() || !strings.HasSuffix(name, ".jsonl") {
				continue
			}
			info, err := parseTranscript(filepath.Join(dir, name))
			if err != nil || info == nil {
				continue
			}
			info.UUID = strings.TrimSuffix(name, ".jsonl")
			out = append(out, *info)
		}
	}
	return out, nil
}

// parseTranscript reads the head of a transcript. Returns nil for sidechains.
func parseTranscript(path string) (*session.TranscriptInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info := &session.TranscriptInfo{Path: path, LastActive: fi.ModTime()}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024) // transcript lines can be large

	gotMeta := false
	for sc.Scan() {
		var ln rawLine
		if err := json.Unmarshal(sc.Bytes(), &ln); err != nil {
			continue // skip malformed lines
		}
		if ln.IsSidechain {
			return nil, nil // skip sub-agent sidechain transcripts
		}
		if !gotMeta && ln.Cwd != "" {
			info.Cwd = ln.Cwd
			info.GitBranch = ln.GitBranch
			gotMeta = true
		}
		if info.FirstUserMsg == "" {
			if msg := firstUserText(ln); msg != "" {
				info.FirstUserMsg = msg
			}
		}
		if gotMeta && info.FirstUserMsg != "" {
			break
		}
	}
	return info, sc.Err()
}

// firstUserText returns a plain-string user prompt, or "" if the line is not a
// real user message (meta, tool result, or structured/array content).
func firstUserText(ln rawLine) string {
	if ln.Type != "user" || ln.IsMeta || len(ln.ToolUseResult) > 0 || len(ln.Message) == 0 {
		return ""
	}
	var m rawMessage
	if err := json.Unmarshal(ln.Message, &m); err != nil {
		return ""
	}
	var s string
	if err := json.Unmarshal(m.Content, &s); err != nil {
		return "" // content is an array (tool result etc.), not a prompt
	}
	return s
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/sources/transcripts.go internal/sources/transcripts_test.go
git commit -m "Add transcript scanning and head parsing"
```

---

### Task 5: Live agents source

**Goal:** Run `claude agents --json` behind an injectable runner and parse it into `[]session.LiveInfo`.

**Files:**
- Create: `internal/sources/agents.go`
- Test: `internal/sources/agents_test.go`

**Acceptance Criteria:**
- [ ] `ListAgents` parses `{pid,cwd,kind,startedAt,sessionId}` into `LiveInfo` (`StartedAt = time.UnixMilli`).
- [ ] Empty output (`""` or `[]`) yields an empty slice, not an error.
- [ ] A runner error propagates; `DefaultRunner` returns `ErrClaudeNotFound` when `claude` is absent.

**Verify:** `go test ./internal/sources/ -run TestAgents -v` → PASS.

**Steps:**

- [ ] **Step 1: Write `internal/sources/agents_test.go`**

```go
package sources

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAgentsParse(t *testing.T) {
	const out = `[{"pid":549,"cwd":"/home/user/cc-orchestra","kind":"interactive","startedAt":1781980821067,"sessionId":"7cb82d9f-178a-54aa-922a-f1643a737531"}]`
	run := func(context.Context) ([]byte, error) { return []byte(out), nil }
	got, err := ListAgents(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 agent, got %d", len(got))
	}
	a := got[0]
	if a.UUID != "7cb82d9f-178a-54aa-922a-f1643a737531" || a.PID != 549 || a.Cwd != "/home/user/cc-orchestra" {
		t.Errorf("agent fields wrong: %+v", a)
	}
	if !a.StartedAt.Equal(time.UnixMilli(1781980821067)) {
		t.Errorf("StartedAt = %v", a.StartedAt)
	}
}

func TestAgentsEmpty(t *testing.T) {
	for _, out := range []string{"", "   ", "[]"} {
		run := func(context.Context) ([]byte, error) { return []byte(out), nil }
		got, err := ListAgents(context.Background(), run)
		if err != nil || len(got) != 0 {
			t.Fatalf("out %q: got (%v,%v)", out, got, err)
		}
	}
}

func TestAgentsRunnerError(t *testing.T) {
	want := errors.New("boom")
	run := func(context.Context) ([]byte, error) { return nil, want }
	if _, err := ListAgents(context.Background(), run); !errors.Is(err, want) {
		t.Fatalf("want propagated error, got %v", err)
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Write `internal/sources/agents.go`**

```go
package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"time"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

// ErrClaudeNotFound indicates the `claude` binary is not on PATH.
var ErrClaudeNotFound = errors.New("claude binary not found on PATH")

// Runner returns the raw bytes of `claude agents --json`.
type Runner func(ctx context.Context) ([]byte, error)

type rawAgent struct {
	PID       int    `json:"pid"`
	Cwd       string `json:"cwd"`
	Kind      string `json:"kind"`
	StartedAt int64  `json:"startedAt"`
	SessionID string `json:"sessionId"`
}

// DefaultRunner runs `claude agents --json`.
func DefaultRunner(ctx context.Context) ([]byte, error) {
	bin, err := exec.LookPath("claude")
	if err != nil {
		return nil, ErrClaudeNotFound
	}
	return exec.CommandContext(ctx, bin, "agents", "--json").Output()
}

// ListAgents runs the runner and parses live sessions. Empty output yields an
// empty slice (no running sessions), not an error.
func ListAgents(ctx context.Context, run Runner) ([]session.LiveInfo, error) {
	out, err := run(ctx)
	if err != nil {
		return nil, err
	}
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil, nil
	}
	var raw []rawAgent
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	live := make([]session.LiveInfo, 0, len(raw))
	for _, a := range raw {
		live = append(live, session.LiveInfo{
			UUID:      a.SessionID,
			PID:       a.PID,
			Cwd:       a.Cwd,
			StartedAt: time.UnixMilli(a.StartedAt),
		})
	}
	return live, nil
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/sources/agents.go internal/sources/agents_test.go
git commit -m "Add live agents source via claude agents --json"
```

---

### Task 6: Registry persistence

**Goal:** Load/save our metadata registry with atomic writes and corrupt-file recovery, and expose it as `session.Meta`.

**Files:**
- Create: `internal/registry/registry.go`
- Test: `internal/registry/registry_test.go`

**Acceptance Criteria:**
- [ ] `Load` of a missing file returns an empty registry (version 1, non-nil map).
- [ ] `Save` then `Load` round-trips entries; no `*.tmp` file remains.
- [ ] `Load` of corrupt JSON returns an empty registry AND renames the bad file to `*.corrupt-*`.
- [ ] `Update` mutates an entry, stamps `UpdatedAt`, and persists (visible after reload).
- [ ] `DefaultPath` honors `$XDG_STATE_HOME`, else `~/.local/state/cc-orchestra/registry.json`.

**Verify:** `go test ./internal/registry/ -v` → PASS.

**Steps:**

- [ ] **Step 1: Write `internal/registry/registry_test.go`**

```go
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
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Write `internal/registry/registry.go`**

```go
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
	defer os.Remove(tmpName) // no-op once renamed
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
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
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/registry/registry.go internal/registry/registry_test.go
git commit -m "Add atomic registry persistence with corrupt recovery"
```

---

### Task 7: TUI core — fleet list, navigation, refresh

**Goal:** A Bubble Tea model that renders the merged fleet as a navigable, scrolling list with a header and footer, hides archived sessions (toggle `A`), refreshes on `r`, and quits on `q`. Mode-specific behaviors (filter, rename, pin, archive, hand-off, preview, polling) are wired as stubs that later tasks fill in — so the keymap and `Update`/`View` shells are written once and stay stable.

**Files:**
- Create: `internal/tui/keys.go`
- Create: `internal/tui/model.go`
- Create: `internal/tui/view.go`
- Test: `internal/tui/model_test.go`

**Acceptance Criteria:**
- [ ] `Init` loads the fleet (a `fleetMsg`); `Update` stores it and computes the visible slice.
- [ ] Archived sessions are hidden until `A` toggles them on.
- [ ] `j`/`k` move the cursor and keep it in range; the viewport scrolls to follow.
- [ ] `q` (and `ctrl+c`) return `tea.Quit`.
- [ ] `View` renders a header, one row per visible session (`★`/glyph/name/project/status/age), and a footer.

**Verify:** `go test ./internal/tui/ -run 'TestNav|TestArchived|TestQuit' -v` → PASS.

**Steps:**

- [ ] **Step 1: Add dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/bubbles@latest github.com/charmbracelet/lipgloss@latest
go mod tidy
```

- [ ] **Step 2: Create `internal/tui/keys.go`**

```go
package tui

import "github.com/charmbracelet/bubbles/key"

type keymap struct {
	Up, Down, Enter, Filter, Rename, Pin, Archive, ShowArchived, Open, Refresh, Quit key.Binding
}

func defaultKeys() keymap {
	return keymap{
		Up:           key.NewBinding(key.WithKeys("k", "up")),
		Down:         key.NewBinding(key.WithKeys("j", "down")),
		Enter:        key.NewBinding(key.WithKeys("enter")),
		Filter:       key.NewBinding(key.WithKeys("/")),
		Rename:       key.NewBinding(key.WithKeys("n")),
		Pin:          key.NewBinding(key.WithKeys("p")),
		Archive:      key.NewBinding(key.WithKeys("a")),
		ShowArchived: key.NewBinding(key.WithKeys("A")),
		Open:         key.NewBinding(key.WithKeys("o")),
		Refresh:      key.NewBinding(key.WithKeys("r")),
		Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c")),
	}
}
```

- [ ] **Step 3: Create `internal/tui/model.go`**

```go
package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jfoo1984/cc-orchestra/internal/registry"
	"github.com/jfoo1984/cc-orchestra/internal/session"
)

// Loader returns the current merged fleet. Injected so the TUI can refresh.
type Loader func() ([]session.Session, error)

// Handoff runs a session and returns when control should come back to the TUI.
// Implemented by ExecHandoff in Task 11.
type Handoff interface {
	Run(uuid string) tea.Cmd
}

// fleetMsg carries a freshly loaded fleet into Update.
type fleetMsg struct {
	sessions []session.Session
	err      error
}

type Model struct {
	keys    keymap
	loader  Loader
	reg     *registry.Registry
	handoff Handoff
	now     func() time.Time

	all     []session.Session
	visible []session.Session
	cursor  int
	top     int // scroll offset

	width, height int
	showArchived  bool

	filtering   bool
	filterInput textinput.Model

	renaming    bool
	renameInput textinput.Model

	preview     string
	previewUUID string

	banner   string
	quitting bool
}

func New(loader Loader, reg *registry.Registry, handoff Handoff, now func() time.Time) Model {
	fi := textinput.New()
	fi.Placeholder = "filter…"
	ri := textinput.New()
	ri.Placeholder = "new name…"
	return Model{
		keys:        defaultKeys(),
		loader:      loader,
		reg:         reg,
		handoff:     handoff,
		now:         now,
		filterInput: fi,
		renameInput: ri,
	}
}

func (m Model) Init() tea.Cmd { return m.refreshCmd() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.adjustScroll()
		return m, nil
	case fleetMsg:
		if msg.err != nil {
			m.banner = "load error: " + msg.err.Error()
		} else {
			m.all = msg.sessions
			m.applyVisible()
		}
		return m, m.previewCmd()
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filtering {
		return m.updateFilter(msg)
	}
	if m.renaming {
		return m.updateRename(msg)
	}
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys.Down):
		m.moveCursor(1)
		return m, m.previewCmd()
	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-1)
		return m, m.previewCmd()
	case key.Matches(msg, m.keys.ShowArchived):
		m.showArchived = !m.showArchived
		m.applyVisible()
	case key.Matches(msg, m.keys.Refresh):
		m.banner = ""
		return m, m.refreshCmd()
	case key.Matches(msg, m.keys.Filter):
		return m.startFilter()
	case key.Matches(msg, m.keys.Rename):
		return m.startRename()
	case key.Matches(msg, m.keys.Pin):
		return m.togglePin()
	case key.Matches(msg, m.keys.Archive):
		return m.toggleArchive()
	case key.Matches(msg, m.keys.Enter):
		return m.doHandoff()
	case key.Matches(msg, m.keys.Open):
		return m.openEditor()
	}
	return m, nil
}

// refreshCmd loads the fleet off the Update loop and delivers a fleetMsg.
func (m Model) refreshCmd() tea.Cmd {
	loader := m.loader
	return func() tea.Msg {
		s, err := loader()
		return fleetMsg{sessions: s, err: err}
	}
}

// applyVisible recomputes the visible slice (archived filter; text filter is
// added in Task 8) and clamps the cursor.
func (m *Model) applyVisible() {
	m.visible = m.visible[:0]
	for _, s := range m.all {
		if s.Archived && !m.showArchived {
			continue
		}
		m.visible = append(m.visible, s)
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.adjustScroll()
}

func (m Model) selected() (session.Session, bool) {
	if m.cursor >= 0 && m.cursor < len(m.visible) {
		return m.visible[m.cursor], true
	}
	return session.Session{}, false
}

func (m *Model) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	m.adjustScroll()
}

func (m *Model) adjustScroll() {
	h := m.listHeight()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+h {
		m.top = m.cursor - h + 1
	}
	if m.top < 0 {
		m.top = 0
	}
}

func (m Model) listHeight() int {
	h := m.height - 4 // header (2) + footer (2)
	if h < 1 {
		return 1
	}
	return h
}

// --- Stubs filled in by later tasks (kept here so handleKey/View compile) ---

func (m Model) previewCmd() tea.Cmd                          { return nil } // Task 9
func (m Model) startFilter() (tea.Model, tea.Cmd)            { return m, nil } // Task 8
func (m Model) updateFilter(tea.KeyMsg) (tea.Model, tea.Cmd) { return m, nil } // Task 8
func (m Model) startRename() (tea.Model, tea.Cmd)            { return m, nil } // Task 10
func (m Model) updateRename(tea.KeyMsg) (tea.Model, tea.Cmd) { return m, nil } // Task 10
func (m Model) togglePin() (tea.Model, tea.Cmd)              { return m, nil } // Task 10
func (m Model) toggleArchive() (tea.Model, tea.Cmd)          { return m, nil } // Task 10
func (m Model) doHandoff() (tea.Model, tea.Cmd)              { return m, nil } // Task 11
func (m Model) openEditor() (tea.Model, tea.Cmd)             { return m, nil } // Task 11
```

- [ ] **Step 4: Create `internal/tui/view.go`**

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")
	b.WriteString(m.renderList())
	b.WriteString("\n")
	if p := m.renderPreview(); p != "" {
		b.WriteString(p)
		b.WriteString("\n")
	}
	b.WriteString(m.renderFooter())
	return b.String()
}

func (m Model) renderHeader() string {
	return headerStyle.Render("cc-orchestra") + "  " +
		dimStyle.Render(fmt.Sprintf("%d sessions", len(m.visible)))
}

func (m Model) renderList() string {
	if len(m.visible) == 0 {
		return dimStyle.Render("  no sessions found")
	}
	h := m.listHeight()
	var b strings.Builder
	for i := m.top; i < len(m.visible) && i < m.top+h; i++ {
		b.WriteString(m.renderRow(m.visible[i], i == m.cursor))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) renderRow(s session.Session, sel bool) string {
	pin := " "
	if s.Pinned {
		pin = "★"
	}
	row := fmt.Sprintf("%s %s %-28s %-16s %-5s %s",
		pin, s.Glyph(), truncTo(s.Name(), 28), truncTo(s.Project, 16),
		s.StatusText(), session.Age(s.LastActive, m.now()))
	if sel {
		return selectedStyle.Render("› " + row)
	}
	return "  " + row
}

func (m Model) renderFooter() string {
	if m.banner != "" {
		return dimStyle.Render(m.banner)
	}
	return dimStyle.Render("j/k move · enter open · / filter · n rename · p pin · a archive · A archived · o editor · r refresh · q quit")
}

func (m Model) renderPreview() string { return "" } // implemented in Task 9

func truncTo(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
```

- [ ] **Step 5: Create `internal/tui/model_test.go`**

```go
package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

func fixedNow() time.Time { return time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC) }

func keyMsg(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func loaded(sessions []session.Session) Model {
	m := New(func() ([]session.Session, error) { return sessions, nil }, nil, nil, fixedNow)
	m.height = 24
	out, _ := m.Update(fleetMsg{sessions: sessions})
	return out.(Model)
}

func TestNavAndArchived(t *testing.T) {
	sessions := []session.Session{
		{UUID: "a", Project: "alpha"},
		{UUID: "b", Project: "beta", Archived: true},
		{UUID: "c", Project: "gamma"},
	}
	m := loaded(sessions)
	if len(m.visible) != 2 {
		t.Fatalf("archived hidden: want 2 visible, got %d", len(m.visible))
	}
	out, _ := m.Update(keyMsg("j"))
	m = out.(Model)
	if m.cursor != 1 {
		t.Fatalf("cursor after j = %d, want 1", m.cursor)
	}
	out, _ = m.Update(keyMsg("A"))
	m = out.(Model)
	if len(m.visible) != 3 {
		t.Fatalf("show archived: want 3 visible, got %d", len(m.visible))
	}
}

func TestQuit(t *testing.T) {
	m := loaded(nil)
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q should return a command (tea.Quit)")
	}
}
```

- [ ] **Step 6: Run tests, expect PASS** — `go test ./internal/tui/ -v`.

- [ ] **Step 7: Commit**

```bash
git add internal/tui go.mod go.sum
git commit -m "Add TUI core: fleet list, navigation, refresh"
```

---

### Task 8: Filter mode

**Goal:** `/` enters filter mode (a focused text input); typing fuzzily narrows the list over name|project|first-message; `enter` confirms (keeps the filter), `esc` clears it.

**Files:**
- Modify: `internal/tui/model.go` (replace the `applyVisible`, `startFilter`, `updateFilter` stubs; add `match`/`subsequence`)
- Modify: `internal/tui/view.go` (replace `renderHeader`)
- Test: `internal/tui/filter_test.go`

**Acceptance Criteria:**
- [ ] A non-empty filter narrows `visible` to fuzzy (subsequence) matches.
- [ ] `esc` clears the filter and restores the full list; `enter` keeps the filter but exits input mode.
- [ ] The header shows the filter input while filtering.

**Verify:** `go test ./internal/tui/ -run TestFilter -v` → PASS.

**Steps:**

- [ ] **Step 1: In `model.go`, replace the `applyVisible` method with:**

```go
func (m *Model) applyVisible() {
	q := strings.ToLower(strings.TrimSpace(m.filterInput.Value()))
	m.visible = m.visible[:0]
	for _, s := range m.all {
		if s.Archived && !m.showArchived {
			continue
		}
		if q != "" && !match(q, s) {
			continue
		}
		m.visible = append(m.visible, s)
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.adjustScroll()
}

// match reports whether the query fuzzily matches the session.
func match(query string, s session.Session) bool {
	hay := strings.ToLower(s.Name() + " " + s.Project + " " + s.FirstUserMsg)
	return subsequence(query, hay)
}

func subsequence(needle, hay string) bool {
	nr := []rune(needle)
	i := 0
	for _, c := range hay {
		if i < len(nr) && nr[i] == c {
			i++
		}
	}
	return i == len(nr)
}
```

Add `"strings"` to `model.go`'s imports.

- [ ] **Step 2: Replace the `startFilter` and `updateFilter` stubs with:**

```go
func (m Model) startFilter() (tea.Model, tea.Cmd) {
	m.filtering = true
	m.filterInput.SetValue("")
	cmd := m.filterInput.Focus()
	m.applyVisible()
	return m, cmd
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filterInput.Blur()
		m.filterInput.SetValue("")
		m.applyVisible()
		return m, nil
	case "enter":
		m.filtering = false
		m.filterInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.cursor = 0
	m.applyVisible()
	return m, cmd
}
```

- [ ] **Step 3: In `view.go`, replace `renderHeader` with:**

```go
func (m Model) renderHeader() string {
	left := headerStyle.Render("cc-orchestra")
	if m.filtering {
		return left + "  /" + m.filterInput.View()
	}
	count := dimStyle.Render(fmt.Sprintf("%d sessions", len(m.visible)))
	if q := m.filterInput.Value(); q != "" {
		count += dimStyle.Render("  filter: " + q)
	}
	return left + "  " + count
}
```

- [ ] **Step 4: Create `internal/tui/filter_test.go`**

```go
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
```

Add this helper to `model_test.go`:

```go
import tea "github.com/charmbracelet/bubbletea"

func tea_esc() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEsc} }

func sessionsFixture() []session.Session {
	return []session.Session{
		{UUID: "a", Project: "alpha"},
		{UUID: "b", Project: "beta"},
		{UUID: "c", Project: "gamma"},
	}
}
```

- [ ] **Step 5: Run, expect PASS** — `go test ./internal/tui/ -v`.

- [ ] **Step 6: Commit**

```bash
git add internal/tui
git commit -m "Add TUI filter mode with fuzzy match"
```

---

### Task 9: Preview pane with debounced lazy load

**Goal:** Show a detail pane for the selected session, lazily loading model / token usage / last messages from the transcript ~150ms after the selection settles.

**Files:**
- Modify: `internal/session/session.go` (add `Detail` type)
- Create: `internal/sources/detail.go`
- Test: `internal/sources/detail_test.go`
- Modify: `internal/tui/model.go` (replace `previewCmd` stub; add `previewTickMsg`/`detailMsg` cases to `Update`)
- Modify: `internal/tui/view.go` (replace `renderPreview`; add `renderDetail` + helpers)

**Acceptance Criteria:**
- [ ] `sources.LoadDetail` extracts the last assistant `model`, token usage, last user text, and last assistant text from a transcript.
- [ ] Moving the cursor schedules a debounced (`tea.Tick` 150ms) load; a stale tick (selection changed) is ignored.
- [ ] The preview pane renders the loaded detail for the selected session.

**Verify:** `go test ./internal/sources/ -run TestLoadDetail -v && go test ./internal/tui/ -run TestPreview -v` → PASS.

**Steps:**

- [ ] **Step 1: In `internal/session/session.go`, add the `Detail` type:**

```go
// Detail holds lazily-loaded preview fields for a session.
type Detail struct {
	Model       string
	Tokens      TokenUsage
	LastUserMsg string
	LastAsstMsg string
}
```

- [ ] **Step 2: Create `internal/sources/detail.go`**

```go
package sources

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

// LoadDetail reads a full transcript and returns the latest model, token usage,
// last user prompt, and last assistant text for the preview pane.
func LoadDetail(path string) (session.Detail, error) {
	f, err := os.Open(path)
	if err != nil {
		return session.Detail{}, err
	}
	defer f.Close()

	var d session.Detail
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		var ln rawLine
		if json.Unmarshal(sc.Bytes(), &ln) != nil {
			continue
		}
		switch ln.Type {
		case "user":
			if t := firstUserText(ln); t != "" {
				d.LastUserMsg = t
			}
		case "assistant":
			if model, text, usage, ok := parseAssistant(ln.Message); ok {
				d.Model = model
				if text != "" {
					d.LastAsstMsg = text
				}
				if usage.Total > 0 {
					d.Tokens = usage
				}
			}
		}
	}
	return d, sc.Err()
}

func parseAssistant(raw json.RawMessage) (model, text string, usage session.TokenUsage, ok bool) {
	if len(raw) == 0 {
		return "", "", session.TokenUsage{}, false
	}
	var m struct {
		Model   string          `json:"model"`
		Content json.RawMessage `json:"content"`
		Usage   struct {
			Input  int `json:"input_tokens"`
			Output int `json:"output_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return "", "", session.TokenUsage{}, false
	}
	usage = session.TokenUsage{Input: m.Usage.Input, Output: m.Usage.Output, Total: m.Usage.Input + m.Usage.Output}
	return m.Model, extractText(m.Content), usage, true
}

// extractText handles assistant content that is either a string or an array of
// content blocks; it returns the first text block.
func extractText(content json.RawMessage) string {
	var s string
	if json.Unmarshal(content, &s) == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(content, &blocks) == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}
```

- [ ] **Step 3: Create `internal/sources/detail_test.go`**

```go
package sources

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDetail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "u.jsonl")
	lines := `{"type":"user","message":{"role":"user","content":"first"}}
{"type":"assistant","message":{"role":"assistant","model":"claude-opus-4-8","content":[{"type":"text","text":"hello there"}],"usage":{"input_tokens":12,"output_tokens":5}}}
{"type":"user","message":{"role":"user","content":"second"}}
`
	if err := os.WriteFile(path, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := LoadDetail(path)
	if err != nil {
		t.Fatal(err)
	}
	if d.Model != "claude-opus-4-8" || d.LastAsstMsg != "hello there" {
		t.Errorf("detail model/text wrong: %+v", d)
	}
	if d.LastUserMsg != "second" || d.Tokens.Total != 17 {
		t.Errorf("detail user/tokens wrong: %+v", d)
	}
}
```

- [ ] **Step 4: In `internal/tui/model.go`, replace the `previewCmd` stub and add the message types:**

```go
type previewTickMsg struct{ uuid string }

type detailMsg struct {
	uuid   string
	detail session.Detail
}

// previewCmd debounces a detail load for the current selection.
func (m Model) previewCmd() tea.Cmd {
	s, ok := m.selected()
	if !ok {
		return nil
	}
	uuid := s.UUID
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
		return previewTickMsg{uuid: uuid}
	})
}
```

- [ ] **Step 5: Add two cases to the `Update` type switch in `model.go`** (right after the `fleetMsg` case):

```go
	case previewTickMsg:
		s, ok := m.selected()
		if !ok || s.UUID != msg.uuid || s.TranscriptPath == "" {
			return m, nil // stale debounce or nothing to load
		}
		path, uuid := s.TranscriptPath, s.UUID
		return m, func() tea.Msg {
			d, _ := sources.LoadDetail(path)
			return detailMsg{uuid: uuid, detail: d}
		}
	case detailMsg:
		if cur, ok := m.selected(); ok && cur.UUID == msg.uuid {
			m.preview = renderDetail(cur, msg.detail, m.now())
			m.previewUUID = msg.uuid
		}
		return m, nil
```

Add `"github.com/jfoo1984/cc-orchestra/internal/sources"` to `model.go`'s imports.

- [ ] **Step 6: In `internal/tui/view.go`, replace `renderPreview` and add helpers:**

```go
var previewBox = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), true, false, false, false).
	BorderForeground(lipgloss.Color("240")).
	MarginTop(1).
	PaddingTop(1)

func (m Model) renderPreview() string {
	if m.preview == "" {
		return ""
	}
	return previewBox.Render(m.preview)
}

func renderDetail(s session.Session, d session.Detail, now time.Time) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(s.Name()) + "\n")
	loc := s.Cwd
	if s.GitBranch != "" {
		loc += "  (" + s.GitBranch + ")"
	}
	b.WriteString(dimStyle.Render(loc) + "\n")
	b.WriteString(fmt.Sprintf("%s %s · %s · %s\n", s.Glyph(), s.StatusText(),
		nz(d.Model, "model ?"), session.Age(s.LastActive, now)))
	if d.Tokens.Total > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("tokens: %d in / %d out\n", d.Tokens.Input, d.Tokens.Output)))
	}
	if d.LastUserMsg != "" {
		b.WriteString("\n› " + truncTo(oneLine(d.LastUserMsg), 200) + "\n")
	}
	if d.LastAsstMsg != "" {
		b.WriteString(truncTo(oneLine(d.LastAsstMsg), 200) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func nz(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }
```

Add `"time"` to `view.go`'s imports.

- [ ] **Step 7: Add a preview test to `internal/tui/preview_test.go`**

```go
package tui

import (
	"testing"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

func TestPreviewRendersDetail(t *testing.T) {
	s := session.Session{UUID: "a", Project: "alpha", Cwd: "/p/alpha", Status: session.StatusIdle}
	m := loaded([]session.Session{s})
	out, _ := m.Update(detailMsg{uuid: "a", detail: session.Detail{Model: "claude-opus-4-8"}})
	m = out.(Model)
	if m.preview == "" || m.previewUUID != "a" {
		t.Fatalf("preview not set: %q / %q", m.preview, m.previewUUID)
	}
}
```

- [ ] **Step 8: Run, expect PASS** — `go test ./internal/sources/ ./internal/tui/ -v`.

- [ ] **Step 9: Commit**

```bash
git add internal/session internal/sources internal/tui
git commit -m "Add preview pane with debounced transcript detail load"
```

---

### Task 10: Registry actions — rename, pin, archive

**Goal:** Wire `n` (rename), `p` (pin/unpin), `a` (archive/unarchive) to mutate the registry, persist atomically, and refresh the view.

**Files:**
- Modify: `internal/tui/model.go` (replace `startRename`, `updateRename`, `togglePin`, `toggleArchive` stubs)
- Modify: `internal/tui/view.go` (show the rename prompt in the header)
- Test: `internal/tui/actions_test.go`

**Acceptance Criteria:**
- [ ] `p` toggles `Pinned` in the registry and the merged view; persists to disk.
- [ ] `a` toggles `Archived`; the row then hides unless `A` is on.
- [ ] `n` opens a prompt seeded with the current name; `enter` saves it as the registry display name; `esc` cancels.
- [ ] After each mutation the fleet is reloaded so the new metadata is reflected.

**Verify:** `go test ./internal/tui/ -run TestActions -v` → PASS.

**Steps:**

- [ ] **Step 1: Replace the four stubs in `model.go`:**

```go
func (m Model) togglePin() (tea.Model, tea.Cmd) {
	s, ok := m.selected()
	if !ok || m.reg == nil {
		return m, nil
	}
	if err := m.reg.Update(s.UUID, func(e *registry.Entry) { e.Pinned = !e.Pinned }); err != nil {
		m.banner = "save failed: " + err.Error()
		return m, nil
	}
	return m, m.refreshCmd()
}

func (m Model) toggleArchive() (tea.Model, tea.Cmd) {
	s, ok := m.selected()
	if !ok || m.reg == nil {
		return m, nil
	}
	if err := m.reg.Update(s.UUID, func(e *registry.Entry) { e.Archived = !e.Archived }); err != nil {
		m.banner = "save failed: " + err.Error()
		return m, nil
	}
	return m, m.refreshCmd()
}

func (m Model) startRename() (tea.Model, tea.Cmd) {
	s, ok := m.selected()
	if !ok {
		return m, nil
	}
	m.renaming = true
	m.renameInput.SetValue(s.Name())
	cmd := m.renameInput.Focus()
	return m, cmd
}

func (m Model) updateRename(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.renaming = false
		m.renameInput.Blur()
		return m, nil
	case "enter":
		m.renaming = false
		m.renameInput.Blur()
		s, ok := m.selected()
		if !ok || m.reg == nil {
			return m, nil
		}
		name := strings.TrimSpace(m.renameInput.Value())
		if err := m.reg.Update(s.UUID, func(e *registry.Entry) { e.Name = name }); err != nil {
			m.banner = "save failed: " + err.Error()
			return m, nil
		}
		return m, m.refreshCmd()
	}
	var cmd tea.Cmd
	m.renameInput, cmd = m.renameInput.Update(msg)
	return m, cmd
}
```

- [ ] **Step 2: In `view.go`, update `renderHeader`** to show the rename prompt (replace the function body's first lines so a rename prompt takes priority):

```go
func (m Model) renderHeader() string {
	left := headerStyle.Render("cc-orchestra")
	switch {
	case m.renaming:
		return left + "  rename: " + m.renameInput.View()
	case m.filtering:
		return left + "  /" + m.filterInput.View()
	}
	count := dimStyle.Render(fmt.Sprintf("%d sessions", len(m.visible)))
	if q := m.filterInput.Value(); q != "" {
		count += dimStyle.Render("  filter: " + q)
	}
	return left + "  " + count
}
```

- [ ] **Step 3: Create `internal/tui/actions_test.go`**

```go
package tui

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jfoo1984/cc-orchestra/internal/registry"
	"github.com/jfoo1984/cc-orchestra/internal/session"
)

func modelWithReg(t *testing.T, sessions []session.Session) (Model, *registry.Registry) {
	t.Helper()
	reg, _ := registry.Load(filepath.Join(t.TempDir(), "registry.json"))
	loader := func() ([]session.Session, error) {
		// Re-apply registry metadata so refreshes reflect mutations.
		metas := reg.Metas()
		out := make([]session.Session, 0, len(sessions))
		for _, s := range sessions {
			if mm, ok := metas[s.UUID]; ok {
				s.Pinned, s.Archived, s.DisplayName = mm.Pinned, mm.Archived, mm.DisplayName
			}
			out = append(out, s)
		}
		return out, nil
	}
	m := New(loader, reg, nil, func() time.Time { return fixedNow() })
	m.height = 24
	out, _ := m.Update(fleetMsg{sessions: sessions})
	return out.(Model), reg
}

func TestActionsPinAndRename(t *testing.T) {
	m, reg := modelWithReg(t, []session.Session{{UUID: "a", Project: "alpha"}})

	out, cmd := m.Update(keyMsg("p"))
	m = out.(Model)
	if cmd == nil {
		t.Fatal("pin should trigger a refresh command")
	}
	if !reg.Sessions["a"].Pinned {
		t.Fatal("pin not persisted to registry")
	}

	out, _ = m.Update(keyMsg("n"))
	m = out.(Model)
	for _, r := range "bot" {
		out, _ = m.Update(keyMsg(string(r)))
		m = out.(Model)
	}
	out, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = out.(Model)
	if reg.Sessions["a"].Name != "bot" {
		t.Fatalf("rename not persisted, got %q", reg.Sessions["a"].Name)
	}
}
```

- [ ] **Step 4: Run, expect PASS** — `go test ./internal/tui/ -v`.

- [ ] **Step 5: Commit**

```bash
git add internal/tui
git commit -m "Wire TUI rename/pin/archive to the registry"
```

---

### Task 11: Hand-off and open-in-editor

**Goal:** `enter` suspends the TUI and runs `claude --resume <uuid>` via `tea.ExecProcess` (resuming the TUI on exit); `o` opens the transcript in `$EDITOR`.

**Files:**
- Create: `internal/tui/handoff.go`
- Modify: `internal/tui/model.go` (replace `doHandoff`/`openEditor` stubs; add `handoffDoneMsg` case to `Update`)
- Test: `internal/tui/handoff_test.go`

**Acceptance Criteria:**
- [ ] `ExecHandoff.Run` returns a `tea.ExecProcess` command for `claude --resume <uuid>`, or a `handoffDoneMsg{err}` if `claude` is absent.
- [ ] `enter` on a row invokes `handoff.Run(uuid)` with the selected UUID.
- [ ] After the external process returns, the fleet refreshes.
- [ ] `o` runs `$EDITOR` (default `vi`) on the selected transcript path.

**Verify:** `go test ./internal/tui/ -run TestHandoff -v` → PASS.

**Steps:**

- [ ] **Step 1: Create `internal/tui/handoff.go`**

```go
package tui

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// handoffDoneMsg is delivered after an external process (session or editor) exits.
type handoffDoneMsg struct{ err error }

// ExecHandoff suspends the TUI and runs `claude --resume <uuid>` on the terminal.
type ExecHandoff struct{}

func (ExecHandoff) Run(uuid string) tea.Cmd {
	bin, err := exec.LookPath("claude")
	if err != nil {
		return func() tea.Msg { return handoffDoneMsg{err: err} }
	}
	c := exec.Command(bin, "--resume", uuid)
	return tea.ExecProcess(c, func(err error) tea.Msg { return handoffDoneMsg{err: err} })
}

// editorCmd runs $EDITOR (default vi) on a path, attached to the terminal.
func editorCmd(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	return tea.ExecProcess(exec.Command(editor, path), func(err error) tea.Msg {
		return handoffDoneMsg{err: err}
	})
}
```

- [ ] **Step 2: Replace the `doHandoff` and `openEditor` stubs in `model.go`:**

```go
func (m Model) doHandoff() (tea.Model, tea.Cmd) {
	s, ok := m.selected()
	if !ok || m.handoff == nil {
		return m, nil
	}
	return m, m.handoff.Run(s.UUID)
}

func (m Model) openEditor() (tea.Model, tea.Cmd) {
	s, ok := m.selected()
	if !ok || s.TranscriptPath == "" {
		return m, nil
	}
	return m, editorCmd(s.TranscriptPath)
}
```

- [ ] **Step 3: Add a `handoffDoneMsg` case to the `Update` type switch in `model.go`:**

```go
	case handoffDoneMsg:
		m.banner = ""
		if msg.err != nil {
			m.banner = "session error: " + msg.err.Error()
		}
		return m, m.refreshCmd()
```

- [ ] **Step 4: Create `internal/tui/handoff_test.go`**

```go
package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

type fakeHandoff struct{ got string }

func (f *fakeHandoff) Run(uuid string) tea.Cmd {
	f.got = uuid
	return nil
}

func TestHandoffUsesSelectedUUID(t *testing.T) {
	fh := &fakeHandoff{}
	sessions := []session.Session{{UUID: "u-1", Project: "alpha"}, {UUID: "u-2", Project: "beta"}}
	m := New(func() ([]session.Session, error) { return sessions, nil }, nil, fh, func() time.Time { return fixedNow() })
	m.height = 24
	out, _ := m.Update(fleetMsg{sessions: sessions})
	m = out.(Model)

	out, _ = m.Update(keyMsg("j")) // move to u-2
	m = out.(Model)
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		cmd() // fakeHandoff.Run returns nil, but doHandoff invoked it
	}
	if fh.got != "u-2" {
		t.Fatalf("handoff got %q, want u-2", fh.got)
	}
}
```

- [ ] **Step 5: Run, expect PASS** — `go test ./internal/tui/ -v`.

- [ ] **Step 6: Commit**

```bash
git add internal/tui
git commit -m "Add tea.ExecProcess hand-off and open-in-editor"
```

---

### Task 12: Live polling

**Goal:** Refresh the fleet automatically every 3 seconds so running/busy status stays current.

**Files:**
- Modify: `internal/tui/model.go` (replace `Init`; add `tickMsg` type + case; add `tickEvery`)
- Test: `internal/tui/poll_test.go`

**Acceptance Criteria:**
- [ ] `Init` starts both an initial load and a 3s ticker.
- [ ] A `tickMsg` triggers a refresh and re-arms the ticker.

**Verify:** `go test ./internal/tui/ -run TestPoll -v` → PASS.

**Steps:**

- [ ] **Step 1: Replace `Init` in `model.go` and add the ticker:**

```go
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refreshCmd(), tickEvery())
}

type tickMsg time.Time

func tickEvery() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}
```

- [ ] **Step 2: Add a `tickMsg` case to the `Update` type switch in `model.go`:**

```go
	case tickMsg:
		return m, tea.Batch(m.refreshCmd(), tickEvery())
```

- [ ] **Step 3: Create `internal/tui/poll_test.go`**

```go
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
```

- [ ] **Step 4: Run, expect PASS** — `go test ./internal/tui/ -v`.

- [ ] **Step 5: Commit**

```bash
git add internal/tui
git commit -m "Add 3s live polling to the TUI"
```

---

### Task 13: main wiring, error handling, and README

**Goal:** Assemble the data sources, registry, and TUI in `main.go`; degrade gracefully when `claude` is missing or `claude agents --json` fails; finalize the README. End-to-end runnable binary.

**Files:**
- Modify: `cmd/cc-orchestra/main.go`
- Modify: `internal/tui/model.go` (add `WithBanner`)
- Modify: `README.md`

**Acceptance Criteria:**
- [ ] `cc-orchestra` launches the TUI listing on-disk sessions.
- [ ] If `claude` is not on `PATH`, the TUI still lists transcripts and shows a banner (no crash).
- [ ] If `claude agents --json` errors, the fleet still renders from transcripts.
- [ ] `go build ./... && go vet ./...` clean; `go test ./...` green.

**Verify:** `go build ./... && go vet ./... && go test ./...` → exits 0; then `./bin/cc-orchestra` (manual) shows the fleet.

**Steps:**

- [ ] **Step 1: Add `WithBanner` to `internal/tui/model.go`:**

```go
// WithBanner returns a copy of the model with an initial footer banner set.
func (m Model) WithBanner(s string) Model {
	m.banner = s
	return m
}
```

- [ ] **Step 2: Replace `cmd/cc-orchestra/main.go`:**

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jfoo1984/cc-orchestra/internal/registry"
	"github.com/jfoo1984/cc-orchestra/internal/session"
	"github.com/jfoo1984/cc-orchestra/internal/sources"
	"github.com/jfoo1984/cc-orchestra/internal/tui"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "cc-orchestra:", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := sources.ProjectsRoot()
	if err != nil {
		return fmt.Errorf("locate projects dir: %w", err)
	}
	regPath, err := registry.DefaultPath()
	if err != nil {
		return fmt.Errorf("locate registry: %w", err)
	}
	reg, err := registry.Load(regPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	now := time.Now
	loader := func() ([]session.Session, error) {
		transcripts, err := sources.Scan(root)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		live, lerr := sources.ListAgents(ctx, sources.DefaultRunner)
		if lerr != nil {
			live = nil // degrade: transcripts only (claude missing or call failed)
		}
		return session.Merge(transcripts, live, reg.Metas(), now()), nil
	}

	m := tui.New(loader, reg, tui.ExecHandoff{}, now)
	if _, lookErr := exec.LookPath("claude"); errors.Is(lookErr, exec.ErrNotFound) || lookErr != nil {
		m = m.WithBanner("claude not found on PATH — transcripts only; hand-off disabled")
	}

	_ = version
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
```

- [ ] **Step 3: Finalize `README.md`:**

```markdown
# cc-orchestra

A terminal UI for managing your local Claude Code sessions. Browse every session
on disk as a fleet, see which are running, and jump straight into one — the TUI
suspends, hands the terminal to `claude --resume`, and returns when you exit.

## Install

    go install github.com/jfoo1984/cc-orchestra/cmd/cc-orchestra@latest

## Usage

Run `cc-orchestra`. Keys:

| Key | Action |
|-----|--------|
| `j`/`k` | move · `enter` open · `/` filter |
| `n` rename · `p` pin · `a` archive · `A` show archived |
| `o` open transcript in `$EDITOR` · `r` refresh · `q` quit |

Sessions come from `~/.claude/projects/**/<uuid>.jsonl`; "running" status comes
from `claude agents --json`; your names/pins/archives live in
`~/.local/state/cc-orchestra/registry.json`.

## Design

See `docs/superpowers/specs/2026-06-20-cc-orchestra-design.md`.
```

- [ ] **Step 4: Verify end-to-end and commit:**

```bash
go build ./... && go vet ./... && go test ./...
go build -o bin/cc-orchestra ./cmd/cc-orchestra
git add cmd internal/tui README.md
git commit -m "Wire main, graceful degradation, and finalize README"
```

---

## Self-Review

- **Spec coverage:** §4 sources → Tasks 4 (transcripts), 5 (agents), 6 (registry), 2 (merge). §4.2 status/`BusyThreshold` → Tasks 1–2. §5 TUI (list/sort/filter/keymap/preview) → Tasks 7–9. §6 hand-off (`tea.ExecProcess`, `Handoff` iface) → Task 11. §7 registry atomic writes → Task 6. §8 layout → Task 0. §9 error handling → Tasks 5 (claude-not-found), 6 (corrupt registry), 13 (banner/degrade). §10 testing → tests in every task. §11 housekeeping → Task 0. Covered.
- **Type consistency:** `session.{Session,Status,TokenUsage,TranscriptInfo,LiveInfo,Meta,Detail}` defined in Tasks 1–2/9 and consumed unchanged by `sources`, `registry`, `tui`. `Handoff.Run(uuid) tea.Cmd` defined in Task 7, implemented in Task 11. `Loader`/`fleetMsg`/`handoffDoneMsg`/`detailMsg` consistent across TUI tasks.
- **No placeholders:** every step ships real code and a concrete verify command. TUI stubs introduced in Task 7 are explicitly replaced in Tasks 8–12 (named functions, not "similar to").
- **Note on incrementality:** Tasks 8–12 modify the `Model`/`Update`/`View` from Task 7 (replace named stubs / add explicit `switch` cases). This is intentional and called out in each task; implement TUI tasks in order.

