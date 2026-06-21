package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jfoo1984/cc-orchestra/internal/registry"
	"github.com/jfoo1984/cc-orchestra/internal/session"
)

// Loader returns the current merged fleet. Injected so the TUI can refresh.
type Loader func() ([]session.Session, error)

// Handoff runs a session and returns when control should come back to the TUI.
// Implemented by ExecHandoff in Task 11.
type Handoff interface {
	Run(uuid string) tea.Cmd
}

// fleetMsg carries a freshly loaded fleet into Update.
type fleetMsg struct {
	sessions []session.Session
	err      error
}

type Model struct {
	keys    keymap
	loader  Loader
	reg     *registry.Registry
	handoff Handoff
	now     func() time.Time

	all     []session.Session
	visible []session.Session
	cursor  int
	top     int // scroll offset

	width, height int
	showArchived  bool

	filtering   bool
	filterInput textinput.Model

	renaming    bool
	renameInput textinput.Model

	preview     string
	previewUUID string

	banner   string
	quitting bool
}

func New(loader Loader, reg *registry.Registry, handoff Handoff, now func() time.Time) Model {
	fi := textinput.New()
	fi.Placeholder = "filter…"
	ri := textinput.New()
	ri.Placeholder = "new name…"
	return Model{
		keys:        defaultKeys(),
		loader:      loader,
		reg:         reg,
		handoff:     handoff,
		now:         now,
		filterInput: fi,
		renameInput: ri,
	}
}

func (m Model) Init() tea.Cmd { return m.refreshCmd() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.adjustScroll()
		return m, nil
	case fleetMsg:
		if msg.err != nil {
			m.banner = "load error: " + msg.err.Error()
		} else {
			m.all = msg.sessions
			m.applyVisible()
		}
		return m, m.previewCmd()
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filtering {
		return m.updateFilter(msg)
	}
	if m.renaming {
		return m.updateRename(msg)
	}
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys.Down):
		m.moveCursor(1)
		return m, m.previewCmd()
	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-1)
		return m, m.previewCmd()
	case key.Matches(msg, m.keys.ShowArchived):
		m.showArchived = !m.showArchived
		m.applyVisible()
	case key.Matches(msg, m.keys.Refresh):
		m.banner = ""
		return m, m.refreshCmd()
	case key.Matches(msg, m.keys.Filter):
		return m.startFilter()
	case key.Matches(msg, m.keys.Rename):
		return m.startRename()
	case key.Matches(msg, m.keys.Pin):
		return m.togglePin()
	case key.Matches(msg, m.keys.Archive):
		return m.toggleArchive()
	case key.Matches(msg, m.keys.Enter):
		return m.doHandoff()
	case key.Matches(msg, m.keys.Open):
		return m.openEditor()
	}
	return m, nil
}

// refreshCmd loads the fleet off the Update loop and delivers a fleetMsg.
func (m Model) refreshCmd() tea.Cmd {
	loader := m.loader
	return func() tea.Msg {
		s, err := loader()
		return fleetMsg{sessions: s, err: err}
	}
}

func (m *Model) applyVisible() {
	q := strings.ToLower(strings.TrimSpace(m.filterInput.Value()))
	m.visible = m.visible[:0]
	for _, s := range m.all {
		if s.Archived && !m.showArchived {
			continue
		}
		if q != "" && !match(q, s) {
			continue
		}
		m.visible = append(m.visible, s)
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.adjustScroll()
}

// match reports whether the query fuzzily matches the session.
func match(query string, s session.Session) bool {
	hay := strings.ToLower(s.Name() + " " + s.Project + " " + s.FirstUserMsg)
	return subsequence(query, hay)
}

func subsequence(needle, hay string) bool {
	nr := []rune(needle)
	i := 0
	for _, c := range hay {
		if i < len(nr) && nr[i] == c {
			i++
		}
	}
	return i == len(nr)
}

func (m Model) selected() (session.Session, bool) {
	if m.cursor >= 0 && m.cursor < len(m.visible) {
		return m.visible[m.cursor], true
	}
	return session.Session{}, false
}

func (m *Model) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	m.adjustScroll()
}

func (m *Model) adjustScroll() {
	h := m.listHeight()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+h {
		m.top = m.cursor - h + 1
	}
	if m.top < 0 {
		m.top = 0
	}
}

func (m Model) listHeight() int {
	h := m.height - 4 // header (2) + footer (2)
	if h < 1 {
		return 1
	}
	return h
}

// --- Stubs filled in by later tasks (kept here so handleKey/View compile) ---

func (m Model) previewCmd() tea.Cmd { return nil } // Task 9

func (m Model) startFilter() (tea.Model, tea.Cmd) {
	m.filtering = true
	m.filterInput.SetValue("")
	cmd := m.filterInput.Focus()
	m.applyVisible()
	return m, cmd
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filterInput.Blur()
		m.filterInput.SetValue("")
		m.applyVisible()
		return m, nil
	case "enter":
		m.filtering = false
		m.filterInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.cursor = 0
	m.applyVisible()
	return m, cmd
}

func (m Model) startRename() (tea.Model, tea.Cmd)            { return m, nil } // Task 10
func (m Model) updateRename(tea.KeyMsg) (tea.Model, tea.Cmd) { return m, nil } // Task 10
func (m Model) togglePin() (tea.Model, tea.Cmd)              { return m, nil } // Task 10
func (m Model) toggleArchive() (tea.Model, tea.Cmd)          { return m, nil } // Task 10
func (m Model) doHandoff() (tea.Model, tea.Cmd)              { return m, nil } // Task 11
func (m Model) openEditor() (tea.Model, tea.Cmd)             { return m, nil } // Task 11
