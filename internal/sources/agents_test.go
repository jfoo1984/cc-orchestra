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
