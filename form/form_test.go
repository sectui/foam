package form

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/internal/foamtest"
	"github.com/sectui/foam/notify"
)

type source struct {
	values   map[string]string
	defaults map[string]string
}

func (s *source) Fields() []Field {
	return []Field{
		{ID: "mode", Label: "Mode", Value: s.get("mode"), Choices: []string{"a", "b", "c"}, Help: "how it runs"},
		{ID: "rate", Label: "Rate limit", Value: s.get("rate"), Origin: s.origin("rate")},
		{ID: "name", Label: strings.Repeat("A very long label ", 4), Value: s.get("name")},
	}
}

func (s *source) get(id string) string {
	if v, ok := s.values[id]; ok {
		return v
	}
	return s.defaults[id]
}

func (s *source) origin(id string) string {
	if _, ok := s.values[id]; ok {
		return "request"
	}
	return "config"
}

func (s *source) Set(id, v string) error {
	if id == "rate" && strings.Trim(v, "0123456789") != "" {
		return errors.New("rate must be a number")
	}
	s.values[id] = v
	return nil
}

func (s *source) Reset(id string) error {
	if id == "name" {
		return errors.New("name cannot be reset")
	}
	delete(s.values, id)
	return nil
}

func newForm() (*Model, *source) {
	src := &source{values: map[string]string{}, defaults: map[string]string{"mode": "a", "rate": "0"}}
	return New("Settings", src), src
}

func TestChoices(t *testing.T) {
	m, src := newForm()
	foamtest.Run(t, m.Scope(), "form.next")
	foamtest.Run(t, m.Scope(), "form.next")
	if src.values["mode"] != "c" {
		t.Fatalf("mode = %q", src.values["mode"])
	}
	foamtest.Run(t, m.Scope(), "form.next")
	if src.values["mode"] != "a" {
		t.Errorf("does not wrap around: %q", src.values["mode"])
	}
	foamtest.Run(t, m.Scope(), "form.prev")
	if src.values["mode"] != "c" {
		t.Errorf("prev: %q", src.values["mode"])
	}
	// edit on a choice cycles too
	foamtest.Run(t, m.Scope(), "form.edit")
	if src.values["mode"] != "a" {
		t.Errorf("edit: %q", src.values["mode"])
	}
	foamtest.Run(t, m.Scope(), "form.reset")
	if _, ok := src.values["mode"]; ok {
		t.Error("reset")
	}
}

func TestTyped(t *testing.T) {
	m, src := newForm()
	foamtest.Run(t, m.Scope(), "form.down")
	foamtest.Run(t, m.Scope(), "form.edit")
	if !m.Capturing() {
		t.Fatal("editing does not capture keys")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	foamtest.Type(m.Update, "x5")
	foamtest.Run(t, m.Scope(), "prompt.submit")
	if src.values["rate"] != "" || !strings.Contains(ansi.Strip(m.View(72, 20)), "rate must be a number") {
		t.Fatalf("invalid value: %q\n%s", src.values["rate"], ansi.Strip(m.View(72, 20)))
	}
	foamtest.Run(t, m.Scope(), "form.edit")
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	foamtest.Type(m.Update, "50")
	foamtest.Run(t, m.Scope(), "prompt.submit")
	if src.values["rate"] != "50" || m.Capturing() {
		t.Fatalf("rate = %q", src.values["rate"])
	}
	if v := ansi.Strip(m.View(72, 20)); !strings.Contains(v, "50") || !strings.Contains(v, "request") {
		t.Errorf("view:\n%s", v)
	}
	// cancel keeps the value
	foamtest.Run(t, m.Scope(), "form.edit")
	foamtest.Type(m.Update, "9")
	foamtest.Run(t, m.Scope(), "prompt.cancel")
	if src.values["rate"] != "50" {
		t.Errorf("cancel changed the value: %q", src.values["rate"])
	}
	// a reset that fails is posted
	foamtest.Run(t, m.Scope(), "form.bottom")
	if !foamtest.Has[notify.Msg](foamtest.Run(t, m.Scope(), "form.reset")) {
		t.Error("reset error not posted")
	}
}

func TestView(t *testing.T) {
	m, _ := newForm()
	foamtest.Fits(t, "form", 6, m.View)
	if v := ansi.Strip(m.View(72, 20)); !strings.Contains(v, "‹ a ›") || !strings.Contains(v, "how it runs") || !strings.Contains(v, "config") {
		t.Errorf("view:\n%s", v)
	}
}
