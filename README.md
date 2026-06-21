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
