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
