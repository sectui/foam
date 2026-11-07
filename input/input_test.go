package input

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func press(m *Model, keys ...tea.KeyPressMsg) {
	for _, k := range keys {
		m.Update(k)
	}
}

func TestEditing(t *testing.T) {
	var m Model
	press(&m, tea.KeyPressMsg{Code: 'a', Text: "a"}, tea.KeyPressMsg{Code: 'b', Text: "b"})
	m.Update(tea.PasteMsg{Content: "c\rd"})
	if m.Value() != "abc d" {
		t.Fatalf("value = %q", m.Value())
	}
	press(&m, tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl})
	if m.Value() != "abc " {
		t.Fatalf("after ctrl+w: %q", m.Value())
	}
	press(&m, tea.KeyPressMsg{Code: tea.KeyHome}, tea.KeyPressMsg{Code: tea.KeyDelete})
	if m.Value() != "bc " {
		t.Fatalf("after home, delete: %q", m.Value())
	}
	if m.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl}) {
		t.Fatal("ctrl+x should not be used")
	}
}
