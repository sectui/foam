// Package form is an overlay editing a list of settings: values picked from a
// set of choices, or typed in.
package form

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anotherhadi/foam/commands"
	"github.com/anotherhadi/foam/draw"
	"github.com/anotherhadi/foam/input"
	"github.com/anotherhadi/foam/list"
	"github.com/anotherhadi/foam/notify"
	"github.com/anotherhadi/foam/overlay"
	"github.com/anotherhadi/foam/theme"
)

type Field struct {
	ID    string
	Label string
	Help  string // shown under the form while the field is selected
	Value string
	// Choices are cycled through; without any, the value is typed in.
	Choices []string
	// Origin is shown beside the value, e.g. "config".
	Origin string
}

// Source owns the values: the form reads them from it and writes them back.
type Source interface {
	Fields() []Field
	Set(id, value string) error
	// Reset drops a value of the source's own, so that a default applies.
	Reset(id string) error
}

type Model struct {
	Title string

	src     Source
	fields  []Field
	list    list.Model
	editing bool
	input   input.Model
	err     string
}

func New(title string, src Source) *Model {
	m := &Model{Title: title, src: src, input: input.New("")}
	m.refresh()
	return m
}

func (m *Model) refresh() {
	m.fields = m.src.Fields()
	m.list.SetLen(len(m.fields))
}

func (m *Model) Capturing() bool { return m.editing }

func (m *Model) current() (Field, bool) {
	if len(m.fields) == 0 {
		return Field{}, false
	}
	return m.fields[m.list.Selected()], true
}

func (m *Model) set(value string) tea.Cmd {
	f, ok := m.current()
	if !ok {
		return nil
	}
	m.err = ""
	if err := m.src.Set(f.ID, value); err != nil {
		m.err = err.Error()
		return nil
	}
	m.refresh()
	return nil
}

func (m *Model) cycle(d int) tea.Cmd {
	f, ok := m.current()
	if !ok || len(f.Choices) == 0 {
		return nil
	}
	i := 0
	for k, c := range f.Choices {
		if c == f.Value {
			i = k
			break
		}
	}
	n := len(f.Choices)
	return m.set(f.Choices[((i+d)%n+n)%n])
}

func (m *Model) edit() tea.Cmd {
	f, ok := m.current()
	if !ok {
		return nil
	}
	if len(f.Choices) > 0 {
		return m.cycle(1)
	}
	m.editing = true
	m.input.SetValue(f.Value)
	return nil
}

func (m *Model) reset() tea.Cmd {
	f, ok := m.current()
	if !ok {
		return nil
	}
	if err := m.src.Reset(f.ID); err != nil {
		return notify.Err(err)
	}
	m.refresh()
	return nil
}

func (m *Model) Scope() commands.Scope {
	if m.editing {
		return commands.Scope{Name: m.Title, Commands: []commands.Command{
			{ID: "prompt.submit", Help: "ok", Run: func() tea.Cmd { m.editing = false; return m.set(m.input.Value()) }},
			{ID: "prompt.cancel", Help: "cancel", Run: func() tea.Cmd { m.editing = false; return nil }},
		}}
	}
	return commands.Scope{Name: m.Title, Commands: []commands.Command{
		{ID: "form.close", Help: "close", Run: func() tea.Cmd { return overlay.Close }},
		{ID: "form.down", Help: "move", Run: func() tea.Cmd { m.list.Move(1); m.err = ""; return nil }},
		{ID: "form.up", Run: func() tea.Cmd { m.list.Move(-1); m.err = ""; return nil }},
		{ID: "form.top", Run: func() tea.Cmd { m.list.Select(0); return nil }},
		{ID: "form.bottom", Run: func() tea.Cmd { m.list.Select(len(m.fields) - 1); return nil }},
		{ID: "form.next", Help: "change", Run: func() tea.Cmd { return m.cycle(1) }},
		{ID: "form.prev", Run: func() tea.Cmd { return m.cycle(-1) }},
		{ID: "form.edit", Help: "edit", Run: m.edit},
		{ID: "form.reset", Help: "default", Run: m.reset},
	}}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if m.editing {
		m.input.Update(msg)
		return nil
	}
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		mm := msg.Mouse()
		mm.Y--
		if i := m.list.Mouse(tea.MouseClickMsg(mm)); i >= 0 {
			return m.edit()
		}
	case tea.MouseWheelMsg:
		m.list.Mouse(msg)
	}
	return nil
}

func (m *Model) View(maxW, maxH int) string {
	w := min(72, maxW)
	inner := max(w-2, 0)
	lw := 0
	for _, f := range m.fields {
		lw = max(lw, ansi.StringWidth(f.Label))
	}
	lw = min(lw, inner/2)
	m.list.SetHeight(max(min(len(m.fields), maxH-6), 1))

	var lines []string
	start, end := m.list.Visible()
	for i := start; i < end; i++ {
		f := m.fields[i]
		sel := i == m.list.Selected()
		label := " " + draw.Pad(ansi.Truncate(f.Label, lw, "…"), lw) + "  "
		vw := max(inner-ansi.StringWidth(label)-1, 1)
		var value string
		switch {
		case sel && m.editing:
			value = m.input.View(vw, true)
		default:
			v := draw.Clean(f.Value)
			if v == "" {
				v = "-"
			}
			if len(f.Choices) > 0 {
				v = "‹ " + v + " ›"
			}
			value = ansi.Truncate(v, vw, "…")
			if o := f.Origin; o != "" && ansi.StringWidth(value)+len(o)+3 <= vw {
				value += theme.Subtle().Render("  " + o)
			}
		}
		row := label + value
		if sel && !m.editing {
			row = draw.Selected(row, inner, true)
		} else {
			row = theme.Subtle().Render(label) + value
		}
		lines = append(lines, row)
	}

	help := ""
	if f, ok := m.current(); ok {
		help = f.Help
	}
	note := lipgloss.NewStyle().Width(inner - 2).Foreground(theme.Subtle().Value())
	if m.err != "" {
		help = m.err
		note = note.Foreground(theme.Danger().Value())
	}
	if help != "" {
		lines = append(lines, draw.Rule(inner))
		for _, l := range strings.Split(note.Render(help), "\n") {
			lines = append(lines, " "+l)
		}
	}
	body := strings.Join(lines, "\n")
	return draw.Frame(body, w, len(lines)+2, draw.FrameOpts{
		Title: m.Title, Active: true, Scroll: m.list.Scroll(),
	})
}
