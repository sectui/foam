package viewer

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestWrapAndSearch(t *testing.T) {
	m := Model{Wrap: true}
	m.SetSize(4, 2)
	m.SetContent([]string{"abcdefghij", "x", "needle here"}, nil)
	if n := m.Scroll().Total; n != 3+1+3 {
		t.Fatalf("rows = %d", n)
	}
	if got := ansi.Strip(m.View()); got != "abcd\nefgh" {
		t.Fatalf("view = %q", got)
	}
	if n := m.Search("NEEDLE"); n != 1 {
		t.Fatalf("matches = %d", n)
	}
	if !strings.Contains(ansi.Strip(m.View()), "need") {
		t.Fatalf("match not revealed: %q", ansi.Strip(m.View()))
	}
}

func TestSanitize(t *testing.T) {
	if got := Sanitize("a\tb\r\x01"); got != "a   b·" {
		t.Fatalf("got %q", got)
	}
}

func TestSplitRows(t *testing.T) {
	styled := "\x1b[31mabcdef\x1b[mgh一二"
	rows := splitRows(styled, 3)
	var plain []string
	for _, r := range rows {
		plain = append(plain, ansi.Strip(r))
	}
	if strings.Join(plain, "|") != "abc|def|gh|一|二" {
		t.Fatalf("rows = %q", plain)
	}
	if !strings.HasPrefix(rows[1], "\x1b[31m") {
		t.Fatalf("style not carried: %q", rows[1])
	}
}
