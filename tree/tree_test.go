package tree

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anotherhadi/foam/theme"
)

type entry struct {
	name string
	dir  bool
}

func key(e entry) (string, bool) { return e.name, e.dir }

func run(m *Model[entry], id string) tea.Msg {
	for _, c := range m.Commands() {
		if c.ID == id {
			if cmd := c.Run(); cmd != nil {
				return cmd()
			}
			return nil
		}
	}
	panic("no command " + id)
}

func view(m *Model[entry]) string { return ansi.Strip(m.View(40, 10, true)) }

func TestTree(t *testing.T) {
	m := New(key)
	m.SetEntries([]entry{{"api", true}, {"api/users", false}, {"login", false}})
	if v := view(m); strings.Contains(v, "users") || !strings.Contains(v, "login") {
		t.Fatalf("folders start closed:\n%s", v)
	}
	run(m, "tree.open") // the folder
	if !strings.Contains(view(m), "users") {
		t.Fatal("open toggles a folder")
	}
	run(m, "list.down")
	if msg, ok := run(m, "tree.open").(OpenMsg); !ok || msg.Name != "api/users" {
		t.Fatalf("open on a leaf: %v", msg)
	}
	if m.Dir() != "api" {
		t.Fatalf("Dir = %q", m.Dir())
	}
	run(m, "tree.collapse")
	if m.SelectedName() != "api" || strings.Contains(view(m), "users") {
		t.Fatal("collapse on a leaf goes to its folder")
	}
	m.SetCurrent("api/users")
	if m.SelectedName() != "api/users" {
		t.Fatal("SetCurrent reveals the entry")
	}
}

func TestFilter(t *testing.T) {
	m := New(key)
	m.SetEntries([]entry{{"a", true}, {"a/b", true}, {"a/b/deep", false}, {"other", false}})
	run(m, "tree.filter")
	for _, r := range "deep" {
		m.Input(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	v := view(m)
	if !strings.Contains(v, "deep") || strings.Contains(v, "other") {
		t.Fatalf("filter:\n%s", v)
	}
	if m.SelectedName() != "a/b/deep" {
		t.Fatalf("the first match is selected, got %q", m.SelectedName())
	}
	m.Input(tea.KeyPressMsg{Code: tea.KeyEnter})
	run(m, "tree.clear-filter")
	if !strings.Contains(view(m), "other") {
		t.Fatal("clear-filter")
	}
}

func TestDot(t *testing.T) {
	m := New(key)
	m.Leaf = func(e entry, current bool) string { return "<" + e.name + ">" }
	m.Dot = func(e entry) (theme.Color, bool) { return theme.Danger(), e.name == "red" }
	m.SetEntries([]entry{{"red", false}, {"plain", false}})
	lines := strings.Split(view(m), "\n")
	if !strings.HasPrefix(lines[0], " ● <red>") && !strings.HasPrefix(lines[0], "  <red>") {
		t.Errorf("dot: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "   <plain>") {
		t.Errorf("room for the dot: %q", lines[1])
	}
}
