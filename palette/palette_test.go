package palette

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/commands"
	"github.com/sectui/foam/internal/foamtest"
)

type ranMsg string

func bindings() []commands.Binding {
	cmd := func(id, title string) commands.Binding {
		return commands.Binding{Command: commands.Command{ID: id, Title: title, Run: func() tea.Cmd {
			return func() tea.Msg { return ranMsg(id) }
		}}}
	}
	return []commands.Binding{
		cmd("app.quit", "Quit"), cmd("run.start", "Run the request"), cmd("settings.open", "Settings"),
		cmd("app.palette", "Command palette"), {Command: commands.Command{ID: "app.hidden"}},
	}
}

func TestFilterAndRun(t *testing.T) {
	m := New(bindings(), "app.palette")
	v := ansi.Strip(m.View(64, 20))
	if strings.Contains(v, "Command palette") || !strings.Contains(v, "Quit") {
		t.Fatalf("skip and titles:\n%s", v)
	}
	foamtest.Type(m.Update, "set")
	msgs := foamtest.Run(t, m.Scope(), "palette.run")
	if len(msgs) == 0 || msgs[len(msgs)-1] != ranMsg("settings.open") {
		t.Fatalf("ran %v", msgs)
	}
	foamtest.Type(m.Update, "zzzz")
	if foamtest.Has[ranMsg](foamtest.Run(t, m.Scope(), "palette.run")) {
		t.Error("ran with no match")
	}
	if !strings.Contains(ansi.Strip(m.View(64, 20)), "no command") {
		t.Error("no hint")
	}
}

func TestMove(t *testing.T) {
	m := New(bindings())
	foamtest.Run(t, m.Scope(), "palette.down")
	msgs := foamtest.Run(t, m.Scope(), "palette.run")
	if msgs[len(msgs)-1] != ranMsg("run.start") {
		t.Fatalf("ran %v", msgs)
	}
	foamtest.Run(t, m.Scope(), "palette.up")
	foamtest.Run(t, m.Scope(), "palette.up")
	if msgs := foamtest.Run(t, m.Scope(), "palette.run"); msgs[len(msgs)-1] != ranMsg("app.quit") {
		t.Fatalf("up past the top: %v", msgs)
	}
}

func TestView(t *testing.T) {
	foamtest.Fits(t, "palette", 5, New(bindings()).View)
}
