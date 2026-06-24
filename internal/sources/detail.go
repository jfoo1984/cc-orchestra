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
	defer func() { _ = f.Close() }()

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
