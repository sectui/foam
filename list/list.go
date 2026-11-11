// Package list keeps a selection and a scroll offset over n rows; the app
// renders the rows themselves.
package list

import (
	tea "charm.land/bubbletea/v2"

	"github.com/sectui/foam/commands"
	"github.com/sectui/foam/draw"
)

type Model struct {
	n, sel, off, h int
}

func (m *Model) SetLen(n int) {
	m.n = max(n, 0)
	m.sel = min(m.sel, max(m.n-1, 0))
	m.clamp()
}

func (m *Model) Len() int { return m.n }

func (m *Model) SetHeight(h int) {
	m.h = max(h, 0)
	m.clamp()
}

func (m *Model) Selected() int { return m.sel }

func (m *Model) Select(i int) {
	m.sel = min(max(i, 0), max(m.n-1, 0))
	m.clamp()
}

func (m *Model) Move(d int) { m.Select(m.sel + d) }

func (m *Model) page() int { return max(m.h-1, 1) }

func (m *Model) clamp() {
	if m.sel < m.off {
		m.off = m.sel
	}
	if m.h > 0 && m.sel >= m.off+m.h {
		m.off = m.sel - m.h + 1
	}
	m.off = max(min(m.off, m.n-m.h), 0)
}

// Visible returns the rows [start, end) on screen.
func (m *Model) Visible() (start, end int) { return m.off, min(m.off+m.h, m.n) }

func (m *Model) Scroll() draw.Scroll { return draw.Scroll{Total: m.n, Visible: m.h, Offset: m.off} }

// Commands are bound under list.*.
func (m *Model) Commands() []commands.Command {
	return []commands.Command{
		{ID: "list.down", Help: "move", Run: func() tea.Cmd { m.Move(1); return nil }},
		{ID: "list.up", Run: func() tea.Cmd { m.Move(-1); return nil }},
		{ID: "list.page-down", Run: func() tea.Cmd { m.Move(m.page()); return nil }},
		{ID: "list.page-up", Run: func() tea.Cmd { m.Move(-m.page()); return nil }},
		{ID: "list.top", Run: func() tea.Cmd { m.Select(0); return nil }},
		{ID: "list.bottom", Run: func() tea.Cmd { m.Select(m.n - 1); return nil }},
	}
}

// y is relative to the first visible row. It returns the clicked row, or -1.
func (m *Model) Mouse(msg tea.MouseMsg) int {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m.Move(-3)
		case tea.MouseWheelDown:
			m.Move(3)
		}
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if i := m.off + msg.Y; msg.Y >= 0 && i < m.n {
				m.Select(i)
				return i
			}
		}
	}
	return -1
}

func (m *Model) Reselect(msg tea.MouseMsg, focused bool) (row int, activate bool) {
	before := m.sel
	i := m.Mouse(msg)
	return i, i >= 0 && focused && i == before
}
