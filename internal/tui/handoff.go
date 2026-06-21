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
