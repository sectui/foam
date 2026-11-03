package editor

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/vim"
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
