package viewer

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func FuzzView(f *testing.F) {
	f.Add("a\tb\nlong line one two\n一二三四\n\x1b[31mx\n", 5, 3, true, "two", 3)
	f.Fuzz(func(t *testing.T, text string, w, h int, wrap bool, q string, scroll int) {
		w, h = w%300, h%100
		if w < 1 || h < 1 {
			return
		}
		var lines []string
		for _, l := range strings.Split(text, "\n") {
			lines = append(lines, Sanitize(l))
		}
		m := Model{Wrap: wrap}
		m.SetSize(w, h)
		m.SetContent(lines, nil)
		m.Search(q)
		m.Next(1)
		m.Next(-1)
		for _, c := range m.Commands() {
			if strings.HasSuffix(c.ID, "down") || strings.HasSuffix(c.ID, "right") || scroll%2 == 0 {
				c.Run()
			}
		}
		out := strings.Split(m.View(), "\n")
		if len(out) > h {
			t.Fatalf("%d rows for a height of %d", len(out), h)
		}
		for _, l := range out {
			if lw := ansi.StringWidth(l); lw > w {
				t.Fatalf("row %q is %d cells wide, w=%d", ansi.Strip(l), lw, w)
			}
		}
		if cur, n := m.Status(); cur > n || cur < 0 {
			t.Fatalf("status %d/%d", cur, n)
		}
	})
}
