package vim

import "testing"

func TestReadOnlyKeeps(t *testing.T) {
	const text = "foo bar\n  baz (x) \"q\"\n\nend"
	for _, c := range []struct {
		name, keys, want string
		cur              Pos
	}{
		{"motions", "wwjl", "", Pos{2, 0}},
		{"gg G", "Ggg", "", Pos{0, 0}},
		{"yank word", "yw", "foo ", Pos{0, 0}},
		{"yank line", "yy", "foo bar", Pos{0, 0}},
		{"yank to end", "jy$", "  baz (x) \"q\"", Pos{1, 0}},
		{"yank text object", `jyi"`, "q", Pos{1, 11}},
		{"yank in visual", "vlly", "foo", Pos{0, 0}},
		{"yank a visual line", "jVy", "  baz (x) \"q\"", Pos{1, 0}},
		{"find char", "f(", "", Pos{0, 0}}, // no ( on the first line
	} {
		t.Run(c.name, func(t *testing.T) {
			e := New(text)
			e.ReadOnly = true
			feed(e, c.keys)
			if got := e.Text(); got != text {
				t.Fatalf("text changed: %q", got)
			}
			if got, _ := e.TakeYank(); got != c.want {
				t.Errorf("yanked %q, want %q", got, c.want)
			}
			if e.Cursor() != c.cur {
				t.Errorf("cursor %v, want %v", e.Cursor(), c.cur)
			}
		})
	}
}

// TestReadOnlyRefuses types every pair of keys the engine knows, from every
// mode: none of them may change the text or open insert mode.
func TestReadOnlyRefuses(t *testing.T) {
	const text = "foo bar\n  baz (x) \"q\"\n\nend"
	starts := []string{"", "v", "V", "y", "2", "y2", "g", "yi", "d"}
	for _, start := range starts {
		for _, a := range fuzzKeys {
			for _, b := range fuzzKeys {
				e := New(text)
				e.ReadOnly = true
				feed(e, start)
				for _, name := range []string{a, b} {
					typed := name
					if len([]rune(name)) > 1 {
						typed = ""
					}
					e.Key(name, typed)
					if got := e.Text(); got != text {
						t.Fatalf("%q then %q %q changed the text: %q", start, a, b, got)
					}
					if e.Mode() == Insert {
						t.Fatalf("%q then %q %q opened insert mode", start, a, b)
					}
				}
			}
		}
	}
}

func TestReadOnlyPaste(t *testing.T) {
	e := New("a")
	e.ReadOnly = true
	e.Paste("b")
	if got := e.Text(); got != "a" {
		t.Fatalf("paste changed the text: %q", got)
	}
}

func FuzzReadOnly(f *testing.F) {
	f.Add("foo bar\n  baz (x) \"q\"\n\nend", []byte{1, 2, 3, 40, 41, 5})
	f.Fuzz(func(t *testing.T, text string, keys []byte) {
		e := New(text)
		e.ReadOnly = true
		// invalid UTF-8 becomes U+FFFD in the buffer: compare with what it holds
		text = e.Text()
		for i, k := range keys {
			name := fuzzKeys[int(k)%len(fuzzKeys)]
			typed := name
			if name == "space" {
				typed = " "
			} else if len([]rune(name)) > 1 {
				typed = ""
			}
			e.Key(name, typed)
			if got := e.Text(); got != text {
				t.Fatalf("text changed after key %d %q: %q", i, name, got)
			}
			if e.Mode() == Insert {
				t.Fatalf("insert mode after key %d %q", i, name)
			}
		}
	})
}
