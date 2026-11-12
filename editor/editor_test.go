package editor

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anotherhadi/foam/vim"
)

func TestWrapAndFollow(t *testing.T) {
	m := New("abcdefghij\nk\nl\nm")
	m.SetSize(4, 2)
	if got := ansi.Strip(m.View(true)); got != "abcd\nefgh" {
		t.Fatalf("view = %q", got)
	}
	m.Key(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if got := ansi.Strip(m.View(true)); got != "l   \nm " && !strings.HasSuffix(got, "m ") {
		t.Fatalf("after G: %q", got)
	}
	if m.Scroll().Offset == 0 {
		t.Fatal("did not scroll")
	}
}

func TestClickAndInsertCursor(t *testing.T) {
	m := New("abcdefghij\nk")
	m.SetSize(4, 5)
	m.Mouse(tea.MouseClickMsg{Button: tea.MouseLeft, X: 1, Y: 1})
	if c := m.Cursor(); c != (vim.Pos{Line: 0, Col: 5}) {
		t.Fatalf("click → %v", c)
	}
	if _, _, ok := m.CursorPosition(); ok {
		t.Fatal("normal mode draws a block")
	}
	m.Key(tea.KeyPressMsg{Code: 'i', Text: "i"})
	if x, y, ok := m.CursorPosition(); !ok || x != 1 || y != 1 {
		t.Fatalf("insert cursor = %d,%d,%v", x, y, ok)
	}
}

func TestWideCharacterWrap(t *testing.T) {
	m := New("0000000000뭼00")
	m.SetSize(11, 5)
	if got := ansi.Strip(m.View(true)); got != "0000000000\n뭼00 " {
		t.Fatalf("a wide character at the end of a row goes to the next one: %q", got)
	}
	// the cursor on 뭼 and after the last character
	m.Editor.Key("i", "i")
	m.Editor.SetCursor(vim.Pos{Line: 0, Col: 10})
	if x, y, _ := m.CursorPosition(); x != 0 || y != 1 {
		t.Errorf("cursor on the wide character at %d,%d", x, y)
	}
	m.Editor.SetCursor(vim.Pos{Line: 0, Col: 13})
	if x, y, _ := m.CursorPosition(); x != 4 || y != 1 {
		t.Errorf("cursor at the end at %d,%d", x, y)
	}
	// a click right of 0000000000, before the row's end, stays on that row
	m.Mouse(tea.MouseClickMsg{X: 10, Y: 0, Button: tea.MouseLeft})
	if c := m.Cursor(); c.Col != 9 {
		t.Errorf("click at the end of row 0: col %d", c.Col)
	}
	m.Mouse(tea.MouseClickMsg{X: 1, Y: 1, Button: tea.MouseLeft})
	if c := m.Cursor(); c.Col != 10 {
		t.Errorf("click on the right half of 뭼: col %d", c.Col)
	}
}

func TestScrollKeys(t *testing.T) {
	var lines []string
	for i := range 100 {
		lines = append(lines, strings.Repeat(string(rune('a'+i%26)), 3))
	}
	m := New(strings.Join(lines, "\n"))
	m.SetSize(10, 10)

	press := func(name string) {
		if !m.Key(tea.KeyPressMsg{Code: keyCode(name), Mod: tea.ModCtrl}) {
			t.Fatalf("%s was not used", name)
		}
	}
	press("ctrl+d")
	if m.Scroll().Offset != 5 || m.Cursor().Line != 5 {
		t.Fatalf("ctrl+d: offset %d, line %d", m.Scroll().Offset, m.Cursor().Line)
	}
	press("ctrl+f")
	if m.Scroll().Offset != 13 || m.Cursor().Line != 13 {
		t.Fatalf("ctrl+f: offset %d, line %d", m.Scroll().Offset, m.Cursor().Line)
	}
	press("ctrl+u")
	if m.Scroll().Offset != 8 || m.Cursor().Line != 8 {
		t.Fatalf("ctrl+u: offset %d, line %d", m.Scroll().Offset, m.Cursor().Line)
	}
	press("ctrl+b")
	if m.Scroll().Offset != 0 || m.Cursor().Line != 0 {
		t.Fatalf("ctrl+b: offset %d, line %d", m.Scroll().Offset, m.Cursor().Line)
	}
	// at the top and at the bottom, the cursor goes where the view cannot
	press("ctrl+u")
	if m.Scroll().Offset != 0 || m.Cursor().Line != 0 {
		t.Fatalf("ctrl+u at the top: offset %d, line %d", m.Scroll().Offset, m.Cursor().Line)
	}
	for range 30 {
		press("ctrl+d")
	}
	if last := m.LineCount() - 1; m.Cursor().Line != last {
		t.Fatalf("ctrl+d at the bottom: line %d, want %d", m.Cursor().Line, last)
	}
	// the selection follows in visual mode
	m.Key(tea.KeyPressMsg{Code: 'g', Text: "g"})
	m.Key(tea.KeyPressMsg{Code: 'g', Text: "g"})
	m.Key(tea.KeyPressMsg{Code: 'V', Text: "V"})
	press("ctrl+d")
	start, end, ok := m.Selection()
	if !ok || start.Line != 0 || end.Line != 5 {
		t.Fatalf("visual ctrl+d: %v..%v (%v)", start, end, ok)
	}
	// in insert mode they are the engine's
	m.Key(tea.KeyPressMsg{Code: tea.KeyEscape})
	m.Key(tea.KeyPressMsg{Code: 'i', Text: "i"})
	before := m.Scroll().Offset
	m.Key(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if m.Scroll().Offset != before {
		t.Fatal("ctrl+d scrolled in insert mode")
	}
}

// A read-only editor scrolls all the same.
func TestScrollReadOnly(t *testing.T) {
	m := New(strings.Repeat("line\n", 100))
	m.ReadOnly = true
	m.SetSize(10, 10)
	if !m.Key(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}) {
		t.Fatal("ctrl+d was not used")
	}
	if m.Scroll().Offset == 0 {
		t.Fatal("did not scroll")
	}
	if got := m.Text(); got != strings.Repeat("line\n", 100) {
		t.Fatalf("the text changed: %q", got)
	}
}

func TestPageKeys(t *testing.T) {
	m := New(strings.Repeat("line\n", 100))
	m.SetSize(10, 10)
	if !m.Key(tea.KeyPressMsg{Code: tea.KeyPgDown}) {
		t.Fatal("pgdown was not used")
	}
	if m.Scroll().Offset != 8 {
		t.Fatalf("pgdown: offset %d", m.Scroll().Offset)
	}
	if !m.Key(tea.KeyPressMsg{Code: tea.KeyPgUp}) {
		t.Fatal("pgup was not used")
	}
	if m.Scroll().Offset != 0 {
		t.Fatalf("pgup: offset %d", m.Scroll().Offset)
	}
}

func keyCode(name string) rune { return rune(name[len("ctrl+")]) }
