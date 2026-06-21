package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/jfoo1984/cc-orchestra/internal/session"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
	previewBox    = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(lipgloss.Color("240")).
			MarginTop(1).
			PaddingTop(1)
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
	left := headerStyle.Render("cc-orchestra")
	if m.filtering {
		return left + "  /" + m.filterInput.View()
	}
	count := dimStyle.Render(fmt.Sprintf("%d sessions", len(m.visible)))
	if q := m.filterInput.Value(); q != "" {
		count += dimStyle.Render("  filter: " + q)
	}
	return left + "  " + count
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

func (m Model) renderPreview() string {
	if m.preview == "" {
		return ""
	}
	return previewBox.Render(m.preview)
}

func renderDetail(s session.Session, d session.Detail, now time.Time) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(s.Name()) + "\n")
	loc := s.Cwd
	if s.GitBranch != "" {
		loc += "  (" + s.GitBranch + ")"
	}
	b.WriteString(dimStyle.Render(loc) + "\n")
	b.WriteString(fmt.Sprintf("%s %s · %s · %s\n", s.Glyph(), s.StatusText(),
		nz(d.Model, "model ?"), session.Age(s.LastActive, now)))
	if d.Tokens.Total > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("tokens: %d in / %d out\n", d.Tokens.Input, d.Tokens.Output)))
	}
	if d.LastUserMsg != "" {
		b.WriteString("\n› " + truncTo(oneLine(d.LastUserMsg), 200) + "\n")
	}
	if d.LastAsstMsg != "" {
		b.WriteString(truncTo(oneLine(d.LastAsstMsg), 200) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func nz(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func truncTo(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
