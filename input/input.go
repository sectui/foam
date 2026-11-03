// Package input is a one-line text field.
package input

import (
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/keys"
	"github.com/sectui/foam/theme"
)

type Model struct {
	Prompt      string
	Placeholder string

	value  []rune
	cursor int
	off    int
}

func New(prompt string) Model { return Model{Prompt: prompt} }

func (m *Model) Value() string { return string(m.value) }

func (m *Model) SetValue(s string) {
	m.value = []rune(s)
	m.cursor = len(m.value)
	m.off = 0
}

func (m *Model) Reset() { m.SetValue("") }

// Update edits the text; it reports whether msg was used.
func (m *Model) Update(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		m.insert(strings.NewReplacer("\r\n", " ", "\n", " ", "\t", " ").Replace(msg.Content))
		return true
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	return false
}

func (m *Model) key(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "left", "ctrl+b":
		m.cursor = max(m.cursor-1, 0)
	case "right", "ctrl+f":
		m.cursor = min(m.cursor+1, len(m.value))
	case "home", "ctrl+a":
		m.cursor = 0
	case "end", "ctrl+e":
		m.cursor = len(m.value)
	case "alt+left", "alt+b":
		m.cursor = m.wordStart()
	case "alt+right", "alt+f":
		m.cursor = m.wordEnd()
	case "backspace", "ctrl+h":
		if m.cursor > 0 {
			m.value = append(m.value[:m.cursor-1], m.value[m.cursor:]...)
			m.cursor--
		}
	case "delete", "ctrl+d":
		if m.cursor < len(m.value) {
			m.value = append(m.value[:m.cursor], m.value[m.cursor+1:]...)
		}
	case "ctrl+u":
		m.value = m.value[m.cursor:]
		m.cursor = 0
	case "ctrl+k":
		m.value = m.value[:m.cursor]
	case "ctrl+w", "alt+backspace":
		start := m.wordStart()
		m.value = append(m.value[:start], m.value[m.cursor:]...)
		m.cursor = start
	default:
		if !keys.Printable(msg) {
			return false
		}
		m.insert(msg.Key().Text)
	}
	return true
}

func (m *Model) insert(s string) {
	r := []rune(s)
	v := make([]rune, 0, len(m.value)+len(r))
	v = append(v, m.value[:m.cursor]...)
	v = append(v, r...)
	m.value = append(v, m.value[m.cursor:]...)
	m.cursor += len(r)
}

func (m *Model) wordStart() int {
	i := m.cursor
	for i > 0 && unicode.IsSpace(m.value[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(m.value[i-1]) {
		i--
	}
	return i
}

func (m *Model) wordEnd() int {
	i := m.cursor
	for i < len(m.value) && unicode.IsSpace(m.value[i]) {
		i++
	}
	for i < len(m.value) && !unicode.IsSpace(m.value[i]) {
		i++
	}
	return i
}

func (m *Model) View(w int, focused bool) string {
	prompt := theme.Primary().Render(m.Prompt)
	w -= ansi.StringWidth(m.Prompt)
	if w <= 1 {
		return prompt
	}
	if len(m.value) == 0 && !focused {
		return prompt + theme.Muted().Render(ansi.Truncate(m.Placeholder, w, "…"))
	}
	// Keep the cursor in view, one cell left free for it at the end.
	if m.cursor < m.off {
		m.off = m.cursor
	}
	for ansi.StringWidth(string(m.value[m.off:m.cursor])) > w-1 {
		m.off++
	}
	before := string(m.value[m.off:m.cursor])
	var b strings.Builder
	b.WriteString(prompt + theme.Fg().Render(before))
	if !focused {
		b.WriteString(theme.Fg().Render(ansi.Truncate(string(m.value[m.cursor:]), w-ansi.StringWidth(before), "")))
		return b.String()
	}
	at, after := " ", ""
	if m.cursor < len(m.value) {
		at, after = string(m.value[m.cursor]), string(m.value[m.cursor+1:])
	}
	b.WriteString(lipgloss.NewStyle().Reverse(true).Render(at))
	if len(m.value) == 0 && m.Placeholder != "" {
		after = theme.Muted().Render(m.Placeholder)
	} else {
		after = theme.Fg().Render(after)
	}
	b.WriteString(ansi.Truncate(after, max(w-ansi.StringWidth(before)-1, 0), ""))
	return b.String()
}
