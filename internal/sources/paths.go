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
