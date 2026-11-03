package list

import "testing"

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
