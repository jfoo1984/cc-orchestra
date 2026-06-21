package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")
	b.WriteString(m.renderList())
	b.WriteString("\n")
	if p := m.renderPreview(); p != "" {
		b.WriteString(p)
		b.WriteString("\n")
	}
	b.WriteString(m.renderFooter())
	return b.String()
}

func (m Model) renderHeader() string {
	return headerStyle.Render("cc-orchestra") + "  " +
		dimStyle.Render(fmt.Sprintf("%d sessions", len(m.visible)))
}

func (m Model) renderList() string {
	if len(m.visible) == 0 {
		return dimStyle.Render("  no sessions found")
	}
	h := m.listHeight()
	var b strings.Builder
	for i := m.top; i < len(m.visible) && i < m.top+h; i++ {
		b.WriteString(m.renderRow(m.visible[i], i == m.cursor))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) renderRow(s session.Session, sel bool) string {
	pin := " "
	if s.Pinned {
		pin = "★"
	}
	row := fmt.Sprintf("%s %s %-28s %-16s %-5s %s",
		pin, s.Glyph(), truncTo(s.Name(), 28), truncTo(s.Project, 16),
		s.StatusText(), session.Age(s.LastActive, m.now()))
	if sel {
		return selectedStyle.Render("› " + row)
	}
	return "  " + row
}

func (m Model) renderFooter() string {
	if m.banner != "" {
		return dimStyle.Render(m.banner)
	}
	return dimStyle.Render("j/k move · enter open · / filter · n rename · p pin · a archive · A archived · o editor · r refresh · q quit")
}

func (m Model) renderPreview() string { return "" } // implemented in Task 9

func truncTo(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
