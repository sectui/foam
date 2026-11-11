package list

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestScrollFollowsSelection(t *testing.T) {
	var m Model
	m.SetLen(100)
	m.SetHeight(10)
	m.Select(50)
	if s, e := m.Visible(); s != 41 || e != 51 {
		t.Fatalf("visible = %d..%d", s, e)
	}
	m.SetLen(20)
	if m.Selected() != 19 {
		t.Fatalf("selected = %d", m.Selected())
	}
	if s, _ := m.Visible(); s != 10 {
		t.Fatalf("start = %d", s)
	}
}

func TestReselect(t *testing.T) {
	var m Model
	m.SetLen(10)
	m.SetHeight(10)
	click := tea.MouseClickMsg{Y: 0, Button: tea.MouseLeft}

	if _, activate := m.Reselect(click, false); activate {
		t.Fatal("not focused: must not activate")
	}
	if _, activate := m.Reselect(click, true); !activate {
		t.Fatal("focused, same row: must activate")
	}
	m.Select(5)
	if _, activate := m.Reselect(click, true); activate {
		t.Fatal("focused, different row: must only select, not activate")
	}
}
