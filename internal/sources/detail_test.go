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
