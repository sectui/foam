package overlay

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anotherhadi/foam/commands"
)

type box struct{}

func (box) Update(tea.Msg) tea.Cmd { return nil }
func (box) View(int, int) string   { return "+--+\n|ok|\n+--+" }
func (box) Scope() commands.Scope  { return commands.Scope{} }

func TestDrawKeepsBackground(t *testing.T) {
	var s Stack
	s.Handle(Open(box{})())
	c := lipgloss.NewCanvas(20, 9)
	c.Compose(lipgloss.NewCompositor(lipgloss.NewLayer(strings.Repeat(strings.Repeat("x", 20)+"\n", 9))))
	s.Draw(c)
	lines := strings.Split(ansi.Strip(c.Render()), "\n")
	if lines[0] != strings.Repeat("x", 20) {
		t.Fatalf("background lost: %q", lines[0])
	}
	if !strings.Contains(lines[3], "x|ok|x") || s.rect.X != 8 {
		t.Fatalf("box not centered (rect %+v):\n%s", s.rect, strings.Join(lines, "\n"))
	}
}
