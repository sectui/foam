// Package confirm is an overlay asking a yes/no question.
package confirm

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anotherhadi/foam/commands"
	"github.com/anotherhadi/foam/draw"
	"github.com/anotherhadi/foam/overlay"
	"github.com/anotherhadi/foam/theme"
)

type Model struct {
	Title    string
	Question string
	Yes, No  string // button labels
	Danger   bool   // the yes button in the danger color

	onYes tea.Cmd
	yes   bool // the selected button
}

// New asks question; onYes runs once the overlay is closed, if the answer is yes.
func New(title, question string, onYes tea.Cmd) *Model {
	return &Model{Title: title, Question: question, Yes: "Yes", No: "No", onYes: onYes}
}

func (m *Model) answer(yes bool) tea.Cmd {
	if yes {
		return tea.Sequence(overlay.Close, m.onYes)
	}
	return overlay.Close
}

func (m *Model) Scope() commands.Scope {
	return commands.Scope{Name: "Question", Commands: []commands.Command{
		{ID: "confirm.yes", Help: "yes", Run: func() tea.Cmd { return m.answer(true) }},
		{ID: "confirm.no", Help: "no", Run: func() tea.Cmd { return m.answer(false) }},
		{ID: "confirm.left", Run: func() tea.Cmd { m.yes = true; return nil }},
		{ID: "confirm.right", Run: func() tea.Cmd { m.yes = false; return nil }},
		{ID: "confirm.select", Help: "select", Run: func() tea.Cmd { return m.answer(m.yes) }},
	}}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd { return nil }

func (m *Model) View(maxW, maxH int) string {
	w := min(56, maxW)
	inner := max(w-4, 1)
	text := lipgloss.NewStyle().Width(inner).Foreground(theme.Fg().Value()).Render(m.Question)
	// the frame, a blank line and the buttons come first
	if lines := strings.Split(text, "\n"); len(lines) > max(maxH-4, 1) {
		lines = lines[:max(maxH-4, 1)]
		lines[len(lines)-1] = ansi.Truncate(lines[len(lines)-1], max(inner-1, 0), "") + "…"
		text = strings.Join(lines, "\n")
	}

	button := func(label string, on bool, c theme.Color) string {
		st := lipgloss.NewStyle().Padding(0, 2).Foreground(theme.Subtle().Value())
		if on {
			st = st.Background(c.Value()).Foreground(theme.Bg().Value()).Bold(true)
		}
		return st.Render(label)
	}
	yc := theme.Primary()
	if m.Danger {
		yc = theme.Danger()
	}
	buttons := button(m.Yes, m.yes, yc) + "  " + button(m.No, !m.yes, theme.Primary())
	buttons = strings.Repeat(" ", max(inner-lipgloss.Width(buttons), 0)) + buttons

	body := lipgloss.NewStyle().Padding(0, 1).Render(text + "\n\n" + buttons)
	return draw.Frame(body, w, lipgloss.Height(body)+2, draw.FrameOpts{Title: m.Title, Active: true})
}
