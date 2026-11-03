// Package help is an overlay listing the active shortcuts, grouped by scope.
package help

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/commands"
	"github.com/sectui/foam/draw"
	"github.com/sectui/foam/overlay"
	"github.com/sectui/foam/theme"
)

type Model struct {
	groups []commands.Group
	off    int
	rows   int
}

// New lists the groups, deepest scope first, as Registry.Groups returns them.
func New(groups []commands.Group) *Model { return &Model{groups: groups} }

func (m *Model) scroll(d int) tea.Cmd {
	m.off = max(m.off+d, 0)
	return nil
}

func (m *Model) Scope() commands.Scope {
	page := func() int { return max(m.rows-1, 1) }
	return commands.Scope{Name: "Help", Commands: []commands.Command{
		{ID: "help.close", Help: "close", Run: func() tea.Cmd { return overlay.Close }},
		{ID: "help.down", Help: "scroll", Run: func() tea.Cmd { return m.scroll(1) }},
		{ID: "help.up", Run: func() tea.Cmd { return m.scroll(-1) }},
		{ID: "help.page-down", Run: func() tea.Cmd { return m.scroll(page()) }},
		{ID: "help.page-up", Run: func() tea.Cmd { return m.scroll(-page()) }},
		{ID: "help.top", Run: func() tea.Cmd { m.off = 0; return nil }},
		{ID: "help.bottom", Run: func() tea.Cmd { m.off = 1 << 30; return nil }},
	}}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if w, ok := msg.(tea.MouseWheelMsg); ok {
		switch w.Button {
		case tea.MouseWheelUp:
			return m.scroll(-3)
		case tea.MouseWheelDown:
			return m.scroll(3)
		}
	}
	return nil
}

// Label is what a command is called in lists: its title, else its help.
func Label(b commands.Binding) string {
	switch {
	case b.Title != "":
		return b.Title
	case b.Help != "":
		return b.Help
	}
	return b.ID
}

func (m *Model) View(maxW, maxH int) string {
	w := min(76, maxW)
	kw := 0
	for _, g := range m.groups {
		for _, b := range g.Bindings {
			kw = max(kw, ansi.StringWidth(b.Keys.String()))
		}
	}
	kw = min(kw, w/2)
	head := theme.Accent().Style().Bold(true)
	ks, ls := theme.Primary().Style(), theme.Fg().Style()

	var lines []string
	for _, g := range m.groups {
		var rows []string
		for _, b := range g.Bindings {
			if len(b.Keys) == 0 {
				continue
			}
			rows = append(rows, "  "+ks.Render(draw.Pad(ansi.Truncate(b.Keys.String(), kw, "…"), kw))+"  "+ls.Render(Label(b)))
		}
		if len(rows) == 0 {
			continue
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, head.Render(g.Scope))
		lines = append(lines, rows...)
	}
	m.rows = max(min(len(lines), maxH-2), 0)
	m.off = min(m.off, max(len(lines)-m.rows, 0))
	return draw.Frame(strings.Join(lines[m.off:m.off+m.rows], "\n"), w, m.rows+2, draw.FrameOpts{
		Title: "Help", Active: true,
		Scroll: draw.Scroll{Total: len(lines), Visible: m.rows, Offset: m.off},
	})
}
