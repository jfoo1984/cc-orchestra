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
