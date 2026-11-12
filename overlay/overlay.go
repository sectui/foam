// Package overlay stacks modal layers over an app: the top one takes the keys
// and the mouse, and what lies under it is dimmed.
package overlay

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/anotherhadi/foam/commands"
	"github.com/anotherhadi/foam/draw"
	"github.com/anotherhadi/foam/geom"
	"github.com/anotherhadi/foam/keys"
)

// Overlay is a modal layer. Its scope replaces the app's commands while it is
// on top; keys none of them take, pastes and mouse events (in coordinates
// local to its box) go to Update.
type Overlay interface {
	Update(msg tea.Msg) tea.Cmd
	View(maxW, maxH int) string
	Scope() commands.Scope
}

// Capturer is implemented by overlays holding a text field: while Capturing,
// printable keys go to Update before any command.
type Capturer interface {
	Capturing() bool
}

type openMsg struct{ o Overlay }
type closeMsg struct{}

func Open(o Overlay) tea.Cmd { return func() tea.Msg { return openMsg{o} } }

// Close removes the top overlay. It is a tea.Cmd.
func Close() tea.Msg { return closeMsg{} }

type Stack struct {
	items []Overlay
	rect  geom.Rect // where the top one was drawn
}

// Handle applies Open and Close; ok is false for any other message.
func (s *Stack) Handle(msg tea.Msg) (ok bool) {
	switch msg := msg.(type) {
	case openMsg:
		s.items = append(s.items, msg.o)
		return true
	case closeMsg:
		if n := len(s.items); n > 0 {
			s.items = s.items[:n-1]
		}
		return true
	}
	return false
}

func (s *Stack) Top() Overlay {
	if len(s.items) == 0 {
		return nil
	}
	return s.items[len(s.items)-1]
}

func (s *Stack) Len() int { return len(s.items) }

func (s *Stack) Key(reg *commands.Registry, msg tea.KeyPressMsg) tea.Cmd {
	o := s.Top()
	if o == nil {
		return nil
	}
	if c, ok := o.(Capturer); ok && c.Capturing() && keys.Printable(msg) {
		return o.Update(msg)
	}
	if b, ok := reg.Match([]commands.Scope{o.Scope()}, msg); ok {
		return b.Run()
	}
	return o.Update(msg)
}

// Mouse hands the top overlay an event inside its box, in local coordinates;
// a click outside closes it.
func (s *Stack) Mouse(msg tea.MouseMsg) tea.Cmd {
	o := s.Top()
	if o == nil {
		return nil
	}
	m := msg.Mouse()
	if !s.rect.Contains(m.X, m.Y) {
		if _, click := msg.(tea.MouseClickMsg); click {
			return Close
		}
		return nil
	}
	m.X, m.Y = s.rect.Local(m.X, m.Y)
	switch msg.(type) {
	case tea.MouseClickMsg:
		return o.Update(tea.MouseClickMsg(m))
	case tea.MouseReleaseMsg:
		return o.Update(tea.MouseReleaseMsg(m))
	case tea.MouseWheelMsg:
		return o.Update(tea.MouseWheelMsg(m))
	case tea.MouseMotionMsg:
		return o.Update(tea.MouseMotionMsg(m))
	}
	return nil
}

// Draw dims c and draws the top overlay over it, centered a third of the way
// down.
func (s *Stack) Draw(c *lipgloss.Canvas) {
	o := s.Top()
	if o == nil {
		s.rect = geom.Rect{}
		return
	}
	draw.Dim(c)
	w, h := c.Width(), c.Height()
	v := o.View(max(w-4, 0), max(h-2, 0))
	vw, vh := lipgloss.Width(v), lipgloss.Height(v)
	s.rect = geom.Rect{X: max((w-vw)/2, 0), Y: max((h-vh)/3, 0), W: vw, H: vh}
	// A lone layer would be drawn at the origin over everything: the
	// compositor is what places it.
	c.Compose(lipgloss.NewCompositor(lipgloss.NewLayer(v).X(s.rect.X).Y(s.rect.Y)))
}
