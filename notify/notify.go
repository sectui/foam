// Package notify carries short messages to wherever an app shows them, and
// keeps their history.
package notify

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anotherhadi/foam/draw"
	"github.com/anotherhadi/foam/theme"
)

type Level int

const (
	InfoLevel Level = iota
	SuccessLevel
	WarningLevel
	ErrorLevel
)

// Msg posts a notification; Duration 0 is the level's default.
type Msg struct {
	Level    Level
	Text     string
	Duration time.Duration
}

func Send(l Level, text string) tea.Cmd {
	return func() tea.Msg { return Msg{Level: l, Text: text} }
}

func Info(text string) tea.Cmd    { return Send(InfoLevel, text) }
func Success(text string) tea.Cmd { return Send(SuccessLevel, text) }
func Warning(text string) tea.Cmd { return Send(WarningLevel, text) }
func Error(text string) tea.Cmd   { return Send(ErrorLevel, text) }

// Err posts err, or nothing when it is nil.
func Err(err error) tea.Cmd {
	if err == nil {
		return nil
	}
	return Error(err.Error())
}

type Entry struct {
	ID    int
	Level Level
	Text  string
	At    time.Time
}

type expireMsg struct{ id int }

// Center keeps the live notifications and the last Max of them.
type Center struct {
	Max     int
	history []Entry
	live    []int
	unread  int
	seq     int
}

// Update handles notify messages; ok is false for anything else.
func (c *Center) Update(msg tea.Msg) (cmd tea.Cmd, ok bool) {
	switch msg := msg.(type) {
	case Msg:
		c.seq++
		e := Entry{ID: c.seq, Level: msg.Level, Text: msg.Text, At: time.Now()}
		c.history = append(c.history, e)
		if m := max(c.Max, 1); len(c.history) > m {
			c.history = c.history[len(c.history)-m:]
		}
		c.live = append(c.live, e.ID)
		c.unread++
		d := msg.Duration
		if d == 0 {
			d = 4 * time.Second
			if msg.Level >= WarningLevel {
				d = 8 * time.Second
			}
		}
		return tea.Tick(d, func(time.Time) tea.Msg { return expireMsg{e.ID} }), true
	case expireMsg:
		for i, id := range c.live {
			if id == msg.id {
				c.live = append(c.live[:i], c.live[i+1:]...)
				break
			}
		}
		return nil, true
	}
	return nil, false
}

func (c *Center) Current() (Entry, bool) {
	if len(c.live) == 0 {
		return Entry{}, false
	}
	id := c.live[len(c.live)-1]
	for i := len(c.history) - 1; i >= 0; i-- {
		if c.history[i].ID == id {
			return c.history[i], true
		}
	}
	return Entry{}, false
}

// Live is newest last.
func (c *Center) Live() []Entry {
	var out []Entry
	for _, id := range c.live {
		for _, e := range c.history {
			if e.ID == id {
				out = append(out, e)
			}
		}
	}
	return out
}

func (c *Center) DismissID(id int) {
	for i, l := range c.live {
		if l == id {
			c.live = append(c.live[:i], c.live[i+1:]...)
			return
		}
	}
}

// Toast styles every line whole, so that the background stays even where the
// text wraps.
func (e Entry) Toast(w int) string {
	col, icon := Style(e.Level)
	bg := theme.Surface().Value()
	inner := max(w-4, 1) // border and padding
	textW := max(inner-ansi.StringWidth(icon)-1, 1)
	text := strings.Split(ansi.Wordwrap(draw.Clean(e.Text), textW, " /-"), "\n")
	iconSt := col.Style().Background(bg).Bold(true)
	textSt := lipgloss.NewStyle().Foreground(col.Value()).Background(bg)
	lines := make([]string, len(text))
	for i, t := range text {
		t = ansi.Truncate(t, textW, "")
		lead := strings.Repeat(" ", ansi.StringWidth(icon))
		if i == 0 {
			lead = icon
		}
		lines[i] = iconSt.Render(lead) + textSt.Render(" "+t+strings.Repeat(" ", textW-ansi.StringWidth(t)))
	}
	return lipgloss.NewStyle().
		Border(theme.Border()).
		BorderForeground(col.Value()).
		BorderBackground(bg).
		Background(bg).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

func (c *Center) Dismiss() { c.live = nil }

// History is newest last.
func (c *Center) History() []Entry { return c.history }

// Unread counts the notifications posted since MarkRead.
func (c *Center) Unread() int { return c.unread }

func (c *Center) MarkRead() { c.unread = 0 }

func Style(l Level) (theme.Color, string) {
	switch l {
	case SuccessLevel:
		return theme.Success(), theme.Icon("\uf00c", "✓")
	case WarningLevel:
		return theme.Warning(), theme.Icon("\uf071", "!")
	case ErrorLevel:
		return theme.Danger(), theme.Icon("\uf00d", "✗")
	}
	return theme.Info(), theme.Icon("\uf05a", "i")
}

func (e Entry) Render() string {
	c, icon := Style(e.Level)
	return c.Style().Bold(true).Render(icon) + " " + c.Render(draw.Clean(e.Text))
}
