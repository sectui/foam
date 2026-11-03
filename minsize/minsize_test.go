package minsize

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestView(t *testing.T) {
	s := Size{W: 80, H: 24}
	if s.Fits(79, 30) || s.Fits(100, 23) || !s.Fits(80, 24) {
		t.Fatal("Fits")
	}
	for _, sz := range [][2]int{{60, 10}, {5, 2}, {1, 1}, {79, 23}} {
		lines := strings.Split(s.View(sz[0], sz[1]), "\n")
		if len(lines) != sz[1] {
			t.Fatalf("%v: %d lines", sz, len(lines))
		}
		for _, l := range lines {
			if ansi.StringWidth(l) != sz[0] {
				t.Fatalf("%v: line %q is %d wide", sz, ansi.Strip(l), ansi.StringWidth(l))
			}
		}
	}
	if !strings.Contains(ansi.Strip(s.View(60, 10)), "needs 80 × 24") {
		t.Fatal("no message")
	}
}
