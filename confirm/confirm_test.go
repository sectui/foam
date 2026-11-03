package confirm

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/internal/foamtest"
)

type yesMsg struct{}

func TestAnswers(t *testing.T) {
	onYes := func() tea.Msg { return yesMsg{} }
	m := New("Delete", "Delete it?", onYes)
	if foamtest.Has[yesMsg](foamtest.Run(t, m.Scope(), "confirm.no")) {
		t.Error("no ran onYes")
	}
	if foamtest.Has[yesMsg](foamtest.Run(t, m.Scope(), "confirm.select")) {
		t.Error("select starts on No, but ran onYes")
	}
	foamtest.Run(t, m.Scope(), "confirm.left")
	if !foamtest.Has[yesMsg](foamtest.Run(t, m.Scope(), "confirm.select")) {
		t.Error("select on Yes did not run onYes")
	}
	foamtest.Run(t, m.Scope(), "confirm.right")
	if foamtest.Has[yesMsg](foamtest.Run(t, m.Scope(), "confirm.select")) {
		t.Error("right goes back to No")
	}
	if !foamtest.Has[yesMsg](foamtest.Run(t, m.Scope(), "confirm.yes")) {
		t.Error("yes did not run onYes")
	}
	// a nil onYes just closes
	foamtest.Run(t, New("x", "y", nil).Scope(), "confirm.yes")
}

func TestView(t *testing.T) {
	m := New("A title long enough to be cut somewhere", strings.Repeat("Is this okay? ", 20), nil)
	m.Danger = true
	foamtest.Fits(t, "confirm", 5, m.View)
	if v := ansi.Strip(m.View(60, 20)); !strings.Contains(v, "Is this okay?") || !strings.Contains(v, "Yes") || !strings.Contains(v, "No") {
		t.Errorf("view:\n%s", v)
	}
}
