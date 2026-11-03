// Package palette is an overlay that finds and runs a command by its title.
package palette

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/commands"
	"github.com/sectui/foam/draw"
	"github.com/sectui/foam/fuzzy"
	"github.com/sectui/foam/input"
	"github.com/sectui/foam/list"
	"github.com/sectui/foam/overlay"
	"github.com/sectui/foam/theme"
)

type Model struct {
	all   []commands.Binding
	shown []commands.Binding
	input input.Model
	list  list.Model
}

// New offers every binding that has a Title, except skip.
func New(active []commands.Binding, skip ...string) *Model {
	m := &Model{input: input.New("> ")}
	m.input.Placeholder = "type a command"
outer:
	for _, b := range active {
		if b.Title == "" {
			continue
		}
		for _, id := range skip {
			if b.ID == id {
				continue outer
			}
		}
		m.all = append(m.all, b)
	}
	m.filter()
	return m
}

func (m *Model) Capturing() bool { return true }

func (m *Model) filter() {
	titles := make([]string, len(m.all))
	for i, b := range m.all {
		titles[i] = b.Title
	}
	m.shown = m.shown[:0]
	for _, i := range fuzzy.Filter(m.input.Value(), titles) {
		m.shown = append(m.shown, m.all[i])
	}
	m.list.SetLen(len(m.shown))
	m.list.Select(0)
}

func (m *Model) run() tea.Cmd {
	if len(m.shown) == 0 {
		return nil
	}
	return tea.Sequence(overlay.Close, m.shown[m.list.Selected()].Run())
}

func (m *Model) Scope() commands.Scope {
	return commands.Scope{Name: "Palette", Commands: []commands.Command{
		{ID: "palette.close", Help: "close", Run: func() tea.Cmd { return overlay.Close }},
		{ID: "palette.run", Help: "run", Run: m.run},
		{ID: "palette.up", Run: func() tea.Cmd { m.list.Move(-1); return nil }},
		{ID: "palette.down", Run: func() tea.Cmd { m.list.Move(1); return nil }},
	}}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg, tea.PasteMsg:
		if m.input.Update(msg) {
			m.filter()
		}
	case tea.MouseClickMsg:
		// border, input and rule come before the first row
		mm := msg.Mouse()
		mm.Y -= 3
		if mm.Y >= 0 && m.list.Mouse(tea.MouseClickMsg(mm)) >= 0 {
			return m.run()
		}
	case tea.MouseWheelMsg:
		m.list.Mouse(msg)
	}
	return nil
}

func (m *Model) View(maxW, maxH int) string {
	w := min(64, maxW)
	inner := max(w-2, 0)
	m.list.SetHeight(max(min(12, maxH-4), 0))

	lines := []string{m.input.View(inner, true), draw.Rule(inner)}
	start, end := m.list.Visible()
	for i := start; i < end; i++ {
		b := m.shown[i]
		k := b.Keys.String()
		t := ansi.Truncate(b.Title, max(inner-ansi.StringWidth(k)-3, 0), "…")
		gap := max(inner-ansi.StringWidth(t)-ansi.StringWidth(k)-2, 1)
		row := " " + theme.Fg().Render(t) + strings.Repeat(" ", gap) + theme.Subtle().Render(k) + " "
		if i == m.list.Selected() {
			row = draw.Selected(row, inner, true)
		}
		lines = append(lines, row)
	}
	if len(m.shown) == 0 {
		lines = append(lines, theme.Subtle().Render(" no command"))
	}
	return draw.Frame(strings.Join(lines, "\n"), w, len(lines)+2, draw.FrameOpts{Title: "Commands", Active: true})
}
