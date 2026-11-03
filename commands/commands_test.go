package commands

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func noop() tea.Cmd { return nil }

func TestShadow(t *testing.T) {
	r, err := New(Bindings{"app": map[string]any{"select": "enter"}, "pane": map[string]any{"open": "o,enter"}})
	if err != nil {
		t.Fatal(err)
	}
	path := []Scope{
		{Name: "app", Commands: []Command{{ID: "app.select", Run: noop}}},
		{Name: "pane", Commands: []Command{{ID: "pane.open", Run: noop}}},
	}
	b, ok := r.Match(path, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !ok || b.ID != "pane.open" {
		t.Fatalf("enter → %q, want pane.open (deeper scope shadows)", b.ID)
	}
	act := r.Active(path)
	if len(act) != 2 || len(act[1].Keys) != 0 {
		t.Fatalf("app.select should have lost enter: %+v", act)
	}
}

func TestCheck(t *testing.T) {
	r, err := New(Bindings{"nope": map[string]any{"x": "q"}, "a": map[string]any{"x": "d"}, "b": map[string]any{"x": "d"}})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Check([]Scope{{Commands: []Command{{ID: "a.x"}, {ID: "b.x"}, {ID: "c.x"}}}})
	for _, want := range []string{"already used", "no keys for (use [] for none): keys.c.x"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in %v", want, err)
		}
	}
	r, _ = New(Bindings{"a": map[string]any{"x": []any{}}})
	if err := r.Check([]Scope{{Commands: []Command{{ID: "a.x"}}}}); err != nil {
		t.Fatalf("[] is a valid binding: %v", err)
	}
	if err := r.Check([]Scope{{Commands: []Command{{ID: "a.x"}, {ID: "o.y", Optional: true}}}}); err != nil {
		t.Fatalf("optional commands need no binding: %v", err)
	}
	if err := r.Check([]Scope{{Commands: []Command{{ID: "a.x"}, {ID: "a.x"}}}}); err == nil || !strings.Contains(err.Error(), "defined twice") {
		t.Fatalf("got %v", err)
	}
}

func TestBindingsNumbers(t *testing.T) {
	r, err := New(Bindings{"page": map[string]any{"list": 1, "other": []any{2, "f2"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := r.KeysOf(Command{ID: "page.other"}).String(); got != "2 / f2" {
		t.Fatalf("got %q", got)
	}
}

func TestBindingsErrors(t *testing.T) {
	if _, err := New(Bindings{"a": map[string]any{"b": []any{"ctrl+", 3}}}); err == nil {
		t.Fatal("expected errors")
	}
}
