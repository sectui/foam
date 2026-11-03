package editor

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

const fuzzKeys = "hjklwbeGg$0ixaAoO\x1bdvVy pu"

func FuzzView(f *testing.F) {
	f.Add("foo\tbar\n一二三 x\n\x01\n", 6, 3, []byte("jjlll$"))
	f.Fuzz(func(t *testing.T, text string, w, h int, keys []byte) {
		w, h = w%200, h%60
		if w < 1 || h < 1 {
			return
		}
		// a render per key: long inputs only test the speed of typing
		keys = keys[:min(len(keys), 400)]
		m := New(text)
		m.SetSize(w, h)
		for _, k := range keys {
			r := rune(fuzzKeys[int(k)%len(fuzzKeys)])
			switch r {
			case '\x1b':
				m.Key(tea.KeyPressMsg{Code: tea.KeyEscape})
			default:
				m.Key(tea.KeyPressMsg{Code: r, Text: string(r)})
			}
			out := strings.Split(m.View(true), "\n")
			if len(out) > h {
				t.Fatalf("%d rows for a height of %d", len(out), h)
			}
			for _, l := range out {
				if lw := ansi.StringWidth(l); lw > w {
					t.Fatalf("row %q is %d cells wide, w=%d", ansi.Strip(l), lw, w)
				}
			}
			if x, y, ok := m.CursorPosition(); ok && (x < 0 || x >= w || y < 0 || y >= h) {
				t.Fatalf("cursor at %d,%d in %dx%d", x, y, w, h)
			}
		}
	})
}
