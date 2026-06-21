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
