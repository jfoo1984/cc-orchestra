# cc-orchestra — Design Spec

- **Date:** 2026-06-20
- **Status:** Approved design (brainstorming complete; implementation plan pending)
- **Branch:** `claude/kind-wright-hw651w`
- **Module:** `github.com/jfoo1984/cc-orchestra` · **Binary:** `cc-orchestra`

---

## 1. Problem & Goals

I often run 5+ Claude Code (CLI) sessions across different projects. When I exit
them or restart my machine, I lose track of which sessions existed and which UUID
maps to what. `claude agents` only shows currently-running sessions; transcripts
on disk are not browsable as a fleet.

**Goals (the felt pain):**

1. **Post-restart recovery** — see every session that ever existed on this machine,
   not just the running ones.
2. **Fast switching** — hop between sessions with a couple of keystrokes.
3. **Fleet-wide visibility** — one view of all sessions with status, project, and
   recency.

## 2. Scope

### In scope (MVP)

- **Local sessions only.** No claude.ai/code cloud sessions. No Anthropic API key.
- **Wrap the `claude` CLI**, not a custom API client.
- A single-binary Go TUI that lists sessions, previews them, and hands off into a
  selected session.

### Out of scope (MVP)

- **Cloud (claude.ai/code) session listing.** Possible later via the Managed Agents
  Sessions API (`GET /v1/sessions`, beta header `managed-agents-2026-04-01`).
  Skipped because it requires an API key and isn't the felt pain.
- **Sending messages into a session from the TUI.**
- **tmux orchestration** (parallel windows). The hand-off layer is designed to
  accept it later (§6) but the MVP ships sequential hand-off only.
- **Cross-machine sync.**

## 3. Tech Stack

**Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea).**

Rationale: fast cold start, single-binary distribution, compounds prior Go learning
(TeslaMonitor), agentic-dev friendly. `tea.ExecProcess` is a native primitive for the
hand-off mechanic (§6); `bubbles/textinput` + `bubbles/key` + `lipgloss` cover filter
input, keymap, and styling. The session list is hand-rolled (slice + cursor) for full
control and easy testing.

## 4. Architecture — Data Sources & Unified Model

Three data sources are merged into one `Session` model keyed by **UUID**.

### 4.1 The three sources

1. **Transcripts** — `~/.claude/projects/{encoded-cwd}/{uuid}.jsonl`. Durable truth.
   - For the **list**: `cwd` + first user message from the transcript head (scan past
     `mode`/meta lines to the first real `user` message — see Appendix); last-active
     from file `mtime` (via `stat`).
   - For the **preview**: lazy-load the rest (model, token usage, last user/assistant
     snippets) on selection, debounced ~150ms.
2. **Live agents** — `claude agents --json`. Tells us *which* sessions are running.
   Verified element shape: `{pid, cwd, kind, startedAt (epoch ms), sessionId}` — there
   is **no** status / name / `updated_at` field. Polled every 3s and on `r`.
3. **Registry** — `~/.local/state/cc-orchestra/registry.json`. Our own metadata
   (custom name, pinned, archived, notes). Atomic writes. See §7.

### 4.2 Unified `Session` model

Keyed by UUID; the fleet is the **union** of transcripts and live agents, enriched
by the registry.

```go
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

type Session struct {
    UUID         string
    Cwd          string    // authoritative, from transcript `cwd` field (§4.4)
    Project      string    // basename(Cwd)
    GitBranch    string    // from transcript, optional
    FirstUserMsg string    // first real user message, for list + fuzzy filter

    Status       Status    // derived: running (agents --json) + mtime freshness
    PID          int       // from `claude agents --json` when running, else 0
    StartedAt    time.Time // from agents --json startedAt (epoch ms), when running
    LastActive   time.Time // transcript mtime (the live source has no updated_at)

    // Lazy preview fields (loaded on selection):
    Model        string
    Tokens       TokenUsage
    LastUserMsg  string
    LastAsstMsg  string

    // Registry-derived metadata:
    DisplayName  string    // custom name, optional
    Pinned       bool
    Archived     bool
    Notes        string

    // Provenance:
    HasTranscript  bool
    IsLive         bool
    TranscriptPath string
}
```

### 4.3 Merge rules

- **Name (display):** `registry.DisplayName` → transcript first-user-message
  (truncated) → `basename(Cwd)`.
- **Status:** if the UUID is in `claude agents --json`, it's running — then
  `StatusBusy` when the transcript `mtime` is within `BusyThreshold` (~10s), else
  `StatusIdle`; if not listed, `StatusNotRunning`.
- **Last-active:** transcript file `mtime` (the live source exposes no `updated_at`;
  `startedAt` is session-start, not last activity).
- A UUID present **only** in live but with no transcript yet still appears (brand-new
  session); a UUID present **only** in the registry but with no transcript and no live
  agent is treated as orphaned (kept, eligible for a future `gc`).

### 4.4 Transcript parsing notes (verified against real data)

- **`cwd` comes from the transcript's embedded `cwd` field — NOT from decoding the
  directory name.** The directory encoding replaces `/` with `-`, which is ambiguous
  because path segments can themselves contain `-`. Example: `/home/user/cc-orchestra`
  encodes to `-home-user-cc-orchestra`; naive `-`→`/` decoding would mangle it into
  `/home/user/cc/orchestra`. The decoded dir name is therefore only a last-resort
  fallback when no line can be read.
- The **first real user message** is the first `type:"user"` line with `isMeta != true`,
  `isSidechain == false`, and a string `message.content` (skip `user` lines that carry
  `toolUseResult` — those are tool results, not prompts). See Appendix for the schema.
- **Filter out `isSidechain: true`** transcripts (sub-agent sidechains, not top-level
  sessions).
- Ignore the `{uuid}.ccr-tip.json` sidecar files.
- Lines are heterogeneous; parse line-by-line, best-effort; skip malformed lines.

## 5. TUI Shape

### Layout

- **Header:** title + filter bar.
- **List** (top/left): one row per session.
- **Preview pane** (bottom or right): details for the highlighted session.

### List rows

```
[★] [●] name                    project            status-text        age
 │   │   │                       │                  │                  └ relative last-active
 │   │   │                       │                  └ e.g. "busy", "idle", "—"
 │   │   │                       └ basename(cwd)
 │   │   └ resolved display name
 │   └ status glyph: ● busy · ◐ idle · ○ not-running
 └ pin indicator
```

### Preview pane

`cwd` · model · token usage · last user message snippet · last assistant message
snippet. Populated by the debounced (~150ms) lazy load described in §4.1.

### Sort & filter

- **Sort tiers:** pinned → busy → idle → not-running; **last-active descending within
  each tier.**
- **Archived** sessions are hidden unless `A` toggles them on.
- **Filter (`/`):** fuzzy match over `name | project | first-user-message`.

### Keymap

| Key | Action |
|-----|--------|
| `j` / `k` | navigate down / up |
| `enter` | hand off into the highlighted session (§6) |
| `/` | filter |
| `n` | rename (writes registry `DisplayName`) |
| `p` | pin / unpin |
| `a` | archive / unarchive |
| `A` | toggle show-archived |
| `o` | open transcript in `$EDITOR` |
| `r` | refresh (re-poll live + rescan) |
| `?` | help |
| `q` | quit the app |

## 6. Hand-off / Switching Mechanic

### Model: the TUI is a persistent home base

- `enter` = **dive into** the highlighted session.
- exiting the Claude session = **surface back** to the fleet view.
- `q` = **leave the app** entirely.

### Mechanism

Use Bubble Tea's **`tea.ExecProcess`**, which suspends the TUI, runs an external
command attached to the real terminal, and resumes the same TUI afterward with model
state (filter, selection) intact.

1. On `enter`, resolve the `claude` binary via `exec.LookPath("claude")`.
2. Issue `tea.ExecProcess(exec.Command(claudePath, "--resume", uuid), onExit)`.
3. Claude inherits the terminal and runs interactively. **No PTY is allocated** —
   the child shares the controlling terminal directly (this is the "open `$EDITOR`
   from a TUI" pattern).
4. When the session exits, `onExit` fires and the TUI resumes; a refresh is triggered
   so status/recency reflect the just-ended session.

`claude --resume <uuid>` is confirmed against the installed CLI (v2.1.181). Related
flags available if needed later: `-c/--continue`, `--fork-session`, `--session-id`.

### Pluggable `Handoff` interface

```go
// Handoff runs a session and returns when control should come back to the TUI.
type Handoff interface {
    Run(uuid string) tea.Cmd
}
```

- **MVP:** `ExecHandoff` — wraps `tea.ExecProcess`; **blocking** (sequential): the TUI
  is hidden while the session runs.
- **Future:** `TmuxHandoff` — detects `$TMUX` and opens `claude --resume <uuid>` in a
  new tmux window; **non-blocking** (parallel): the TUI stays in the current window.
  Drops in without touching the data layer.

### Concurrency

"Mix" model: usually sequential, occasionally parallel. **MVP is sequential only**
(`ExecHandoff`); the parallel path arrives with `TmuxHandoff` post-MVP.

### Platform

`tea.ExecProcess` with an inherited terminal targets Unix (Linux/macOS). Windows is
out of scope and documented as such.

## 7. Registry Schema

**Path:** `~/.local/state/cc-orchestra/registry.json` (honor `$XDG_STATE_HOME`;
fall back to `~/.local/state`).

```json
{
  "version": 1,
  "sessions": {
    "<uuid>": {
      "name": "optional custom display name",
      "pinned": false,
      "archived": false,
      "notes": "optional free text",
      "updated_at": "2026-06-20T14:39:00Z"
    }
  }
}
```

- Stores **only our metadata** — never duplicates derivable transcript/live data (DRY).
- **Atomic writes:** write `registry.json.tmp` in the same directory → `fsync` →
  `os.Rename` over the target. Create the directory on first write.
- **Concurrency:** single-writer assumption (one TUI). On save, reload-merge-write to
  limit clobber if two instances run; last-write-wins beyond that (acceptable for MVP).
- **Resilience:** a missing registry is treated as empty. A corrupt registry is backed
  up (`registry.json.corrupt-<ts>`), logged, and treated as empty rather than crashing.
- **Pruning** orphaned entries (UUIDs with no transcript) is deferred to a future `gc`.

## 8. Project Layout

```
cc-orchestra/
├── go.mod                       # module github.com/jfoo1984/cc-orchestra (go 1.24)
├── go.sum
├── Makefile                     # build / test / install / lint / run
├── .golangci.yml
├── .github/workflows/ci.yml     # go test + golangci-lint
├── README.md
├── LICENSE                      # MIT (present)
├── .gitignore                   # present
├── cmd/
│   └── cc-orchestra/
│       └── main.go              # wire sources + registry + tui; run program
└── internal/
    ├── session/                 # unified Session model + merge rules (§4.2–4.3)
    ├── sources/
    │   ├── transcripts.go       # scan ~/.claude/projects, parse JSONL (§4.4)
    │   ├── agents.go            # `claude agents --json` runner + parse
    │   └── paths.go             # projects root + project-dir decode (fallback)
    ├── registry/                # atomic load/save (§7)
    └── tui/                     # model · update · view · keys · handoff
```

## 9. Error Handling

- **No `~/.claude/projects`** → empty fleet with a friendly empty-state message.
- **`claude` binary not found** (`LookPath` fails) → disable hand-off and the live
  source; show transcripts-only with a banner explaining why.
- **`claude agents --json` errors / times out** → keep last-known live state, mark it
  stale, never crash.
- **Malformed JSONL lines** → skip and continue (best-effort parse).
- **Corrupt registry** → back up + treat as empty (§7).
- **Atomic write failure** → surface a non-fatal error; keep in-memory state.

## 10. Testing Strategy

- **Pure logic, table-driven unit tests:**
  - `sources/paths`: dir decode fallback + basename.
  - `sources/transcripts`: first-user-message extraction (skipping meta/tool-result
    lines), `cwd`/`gitBranch`/`isSidechain` parsing, mtime, malformed-line skipping —
    driven by `testdata/*.jsonl` fixtures.
  - `sources/agents`: parse canned `claude agents --json` output via a fake runner.
  - `registry`: atomic round-trip, corrupt-file recovery.
  - `session`: merge precedence (registry ∪ transcript ∪ live), status derivation
    (busy/idle via `BusyThreshold`), sort, fuzzy filter.
- **Seams for isolation (no real side effects in tests):**
  - The `claude agents --json` runner is a function/interface var so tests inject canned
    JSON instead of shelling out.
  - `Handoff` behind its interface so tests assert the selected UUID without launching
    Claude.
- **TUI:** drive `Update` with synthetic messages and assert model state.

## 11. Repo Housekeeping Decisions

- **Naming:** `cc-orchestra` everywhere — module `github.com/jfoo1984/cc-orchestra`,
  binary `cc-orchestra`, registry dir `~/.local/state/cc-orchestra/`.
- **License:** MIT (already committed).
- **Visibility:** public.
- **CI:** day-one GitHub Actions running `go test` + `golangci-lint`.
- **Distribution:** `go install` only for MVP (`go install
  github.com/jfoo1984/cc-orchestra/cmd/cc-orchestra@latest`). goreleaser + brew tap
  deferred.

## 12. Open Questions / Future Work

- tmux parallel hand-off (`TmuxHandoff`).
- Cloud session listing via the Managed Agents Sessions API.
- A reliable "waiting for input" status (not locally detectable today).
- Registry `gc` for orphaned entries.
- goreleaser + brew distribution.

---

## Appendix — Verified CLI / Transcript Contracts

Confirmed against the installed `claude` CLI (v2.1.181) and real on-disk transcripts:

- **Resume:** `-r, --resume [value]` — "Resume a conversation by session ID." So
  `claude --resume <uuid>` is correct.
- **Live agents:** `claude agents --json` — "Print active sessions as a JSON array and
  exit … does not require a TTY." Each element: `{pid, cwd, kind, startedAt (epoch ms),
  sessionId}` — **no status/name/updated_at**. Flags: `--all` (include completed),
  `--cwd <path>` (filter).
- **Transcript path:** `~/.claude/projects/{encoded-cwd}/{uuid}.jsonl`, e.g.
  `~/.claude/projects/-home-user-cc-orchestra/<uuid>.jsonl`.
- **Transcript line schema (verified):** newline-delimited JSON; line `type` ∈
  {`user`, `assistant`, `attachment`, `queue-operation`, `last-prompt`, `mode`}. Every
  message line carries `uuid`, `parentUuid`, `sessionId`, `timestamp`, `cwd`,
  `gitBranch`, `isSidechain`. The **first real user message** = first `type:"user"`
  line with `isMeta != true`, `isSidechain == false`, and a string `message.content`
  (skip `user` lines that carry `toolUseResult`). Assistant lines hold the Anthropic
  message in `message` (with `model` and `usage` for the preview). A
  `{uuid}.ccr-tip.json` sidecar sits alongside each transcript; ignore it.
