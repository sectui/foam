package help

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anotherhadi/foam/commands"
	"github.com/anotherhadi/foam/internal/foamtest"
	"github.com/anotherhadi/foam/keys"
)

func groups(n int) []commands.Group {
	var bs []commands.Binding
	for i := range n {
		bs = append(bs, commands.Binding{
			Command: commands.Command{ID: fmt.Sprintf("app.c%d", i), Title: fmt.Sprintf("Command %d", i)},
			Keys:    keys.List{keys.Key(fmt.Sprintf("ctrl+%c", 'a'+i%26))},
		})
	}
	bs = append(bs, commands.Binding{Command: commands.Command{ID: "app.unbound", Title: "Unbound"}})
	return []commands.Group{{Scope: "Pane", Bindings: bs[:n/2]}, {Scope: "App", Bindings: bs[n/2:]}, {Scope: "Empty"}}
}

func TestView(t *testing.T) {
	m := New(groups(40))
	foamtest.Fits(t, "help", 3, m.View)
	v := ansi.Strip(m.View(80, 100))
	for _, want := range []string{"Pane", "App", "ctrl+a", "Command 0", "Command 39"} {
		if !strings.Contains(v, want) {
			t.Errorf("no %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "Unbound") || strings.Contains(v, "Empty") {
		t.Errorf("unbound commands and empty groups are not listed:\n%s", v)
	}
}

func TestScroll(t *testing.T) {
	m := New(groups(40))
	top := ansi.Strip(m.View(80, 12))
	foamtest.Run(t, m.Scope(), "help.bottom")
	bottom := ansi.Strip(m.View(80, 12))
	if top == bottom || !strings.Contains(bottom, "Command 39") {
		t.Fatalf("bottom:\n%s", bottom)
	}
	// scrolling past the end stays at the end
	foamtest.Run(t, m.Scope(), "help.page-down")
	if ansi.Strip(m.View(80, 12)) != bottom {
		t.Error("scrolled past the end")
	}
	m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	foamtest.Run(t, m.Scope(), "help.top")
	if ansi.Strip(m.View(80, 12)) != top {
		t.Error("top")
	}
	foamtest.Run(t, m.Scope(), "help.up")
	if ansi.Strip(m.View(80, 12)) != top {
		t.Error("scrolled above the top")
	}
}

func TestLabel(t *testing.T) {
	for _, c := range []struct {
		cmd  commands.Command
		want string
	}{
		{commands.Command{ID: "a.b", Title: "T", Help: "h"}, "T"},
		{commands.Command{ID: "a.b", Help: "h"}, "h"},
		{commands.Command{ID: "a.b"}, "b"},
		{commands.Command{ID: "viewer.page-down"}, "page down"},
		{commands.Command{ID: "quit"}, "quit"},
	} {
		if got := Label(commands.Binding{Command: c.cmd}); got != c.want {
			t.Errorf("Label = %q, want %q", got, c.want)
		}
	}
}
