package vim

import "testing"

var fuzzKeys = []string{"h", "j", "k", "l", "w", "b", "e", "W", "B", "E", "0", "^", "$", "g", "G", "{", "}", "f", "t", "F", "T",
	"i", "a", "I", "A", "o", "O", "x", "X", "d", "c", "y", "D", "C", "Y", "p", "P", "J", "r", "~", "u", "ctrl+r",
	"v", "V", "esc", "enter", "backspace", "delete", "tab", "ctrl+w", "ctrl+u", "left", "right", "up", "down", "home", "end",
	"1", "2", "3", "9", "\"", "(", ")", "[", "<", "'", "space", "Z", "é"}

func FuzzEditor(f *testing.F) {
	f.Add("foo bar\n  baz (x) \"q\"\n\nend", []byte{1, 2, 3, 40, 41, 5})
	f.Fuzz(func(t *testing.T, text string, keys []byte) {
		e := New(text)
		for i, k := range keys {
			name := fuzzKeys[int(k)%len(fuzzKeys)]
			typed := name
			if name == "space" {
				typed = " "
			} else if len([]rune(name)) > 1 {
				typed = ""
			}
			e.Key(name, typed)
			c := e.Cursor()
			if c.Line < 0 || c.Line >= e.LineCount() || c.Col < 0 || c.Col > len([]rune(e.Line(c.Line))) {
				t.Fatalf("cursor %v outside the text after key %d %q", c, i, name)
			}
			if l := len([]rune(e.Line(c.Line))); e.Mode() == Normal && c.Col >= max(l, 1) {
				t.Fatalf("cursor %v past the last character in normal mode after key %d %q", c, i, name)
			}
			if a, b, ok := e.Selection(); ok && (b.less(a) || b.Line >= e.LineCount()) {
				t.Fatalf("bad selection %v %v after key %d %q", a, b, i, name)
			}
		}
	})
}
