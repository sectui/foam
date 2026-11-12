// Package prompt is an overlay asking for one line of text.
package prompt

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/anotherhadi/foam/commands"
	"github.com/anotherhadi/foam/draw"
	"github.com/anotherhadi/foam/input"
	"github.com/anotherhadi/foam/overlay"
	"github.com/anotherhadi/foam/theme"
)

type Model struct {
	Title string
	Hint  string // under the field
	// Validate, when set, is checked as the text changes and before submitting.
	Validate func(string) error

	input    input.Model
	onSubmit func(string) tea.Cmd
	err      error
}

// New asks for a line, starting from value; onSubmit gets it once the overlay
// is closed.
func New(title, value string, onSubmit func(string) tea.Cmd) *Model {
	m := &Model{Title: title, input: input.New("> "), onSubmit: onSubmit}
	m.input.SetValue(value)
	return m
}

func (m *Model) Capturing() bool { return true }

func (m *Model) check() {
	m.err = nil
	if m.Validate != nil {
		m.err = m.Validate(m.input.Value())
	}
}

func (m *Model) submit() tea.Cmd {
	if m.check(); m.err != nil {
		return nil
	}
	return tea.Sequence(overlay.Close, m.onSubmit(m.input.Value()))
}

func (m *Model) Scope() commands.Scope {
	return commands.Scope{Name: m.Title, Commands: []commands.Command{
		{ID: "prompt.submit", Help: "ok", Run: m.submit},
		{ID: "prompt.cancel", Help: "cancel", Run: func() tea.Cmd { return overlay.Close }},
	}}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if m.input.Update(msg) {
		m.check()
	}
	return nil
}

func (m *Model) View(maxW, maxH int) string {
	w := min(64, maxW)
	inner := max(w-2, 0)
	lines := []string{m.input.View(inner, true)}
	switch {
	case m.err != nil:
		lines = append(lines, lipgloss.NewStyle().Width(inner).Foreground(theme.Danger().Value()).Render(m.err.Error()))
	case m.Hint != "":
		lines = append(lines, lipgloss.NewStyle().Width(inner).Foreground(theme.Subtle().Value()).Render(m.Hint))
	}
	body := strings.Join(lines, "\n")
	// the frame and the field come first
	if rows := strings.Split(body, "\n"); len(rows) > max(maxH-2, 1) {
		body = strings.Join(rows[:max(maxH-2, 1)], "\n")
	}
	return draw.Frame(body, w, lipgloss.Height(body)+2, draw.FrameOpts{Title: m.Title, Active: true})
}
