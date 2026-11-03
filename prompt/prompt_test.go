package prompt

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/internal/foamtest"
)

type gotMsg string

func TestSubmit(t *testing.T) {
	m := New("Name", "api/", func(s string) tea.Cmd { return func() tea.Msg { return gotMsg(s) } })
	foamtest.Type(m.Update, "login")
	msgs := foamtest.Run(t, m.Scope(), "prompt.submit")
	if !foamtest.Has[gotMsg](msgs) || msgs[len(msgs)-1] != gotMsg("api/login") {
		t.Fatalf("got %v", msgs)
	}
	if foamtest.Has[gotMsg](foamtest.Run(t, m.Scope(), "prompt.cancel")) {
		t.Error("cancel submitted")
	}
}

func TestValidate(t *testing.T) {
	m := New("Name", "", func(s string) tea.Cmd { return func() tea.Msg { return gotMsg(s) } })
	m.Validate = func(s string) error {
		if strings.Contains(s, "/") {
			return errors.New("no slash allowed")
		}
		return nil
	}
	foamtest.Type(m.Update, "a/b")
	if !strings.Contains(ansi.Strip(m.View(60, 10)), "no slash allowed") {
		t.Errorf("error not shown:\n%s", ansi.Strip(m.View(60, 10)))
	}
	if foamtest.Has[gotMsg](foamtest.Run(t, m.Scope(), "prompt.submit")) {
		t.Error("submitted an invalid value")
	}
	// the error goes as the text gets valid
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if strings.Contains(ansi.Strip(m.View(60, 10)), "no slash") {
		t.Error("error kept")
	}
	if !foamtest.Has[gotMsg](foamtest.Run(t, m.Scope(), "prompt.submit")) {
		t.Error("valid value not submitted")
	}
}

func TestView(t *testing.T) {
	m := New("A long title for a prompt", strings.Repeat("value ", 30), nil)
	m.Hint = strings.Repeat("a hint that wraps ", 10)
	foamtest.Fits(t, "prompt", 4, m.View)
	if !m.Capturing() {
		t.Error("a prompt captures printable keys")
	}
}
