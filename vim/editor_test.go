package vim

import (
	"strings"
	"testing"
)

// feed types keys: plain characters, or <name> for a named key (<esc>,
// <enter>, <ctrl+r>).
func feed(e *Editor, keys string) {
	for i := 0; i < len(keys); i++ {
		if keys[i] == '<' {
			if j := strings.IndexByte(keys[i:], '>'); j > 1 {
				e.Key(keys[i+1:i+j], "")
				i += j
				continue
			}
		}
		c := string(keys[i])
		name := c
		if c == " " {
			name = "space"
		}
		e.Key(name, c)
	}
}

func TestCommands(t *testing.T) {
	for _, c := range []struct {
		name, text, keys, want string
		cur                    Pos
	}{
		{"dw", "foo bar baz", "dw", "bar baz", Pos{0, 0}},
		{"d2w", "foo bar baz", "d2w", "baz", Pos{0, 0}},
		{"2dw", "foo bar baz", "2dw", "baz", Pos{0, 0}},
		{"dw last word", "foo bar\nnext", "wdw", "foo \nnext", Pos{0, 3}},
		{"cw keeps space", "foo bar", "cwx<esc>", "x bar", Pos{0, 0}},
		{"ce", "foo.bar baz", "cex<esc>", "x.bar baz", Pos{0, 0}},
		{"dd", "a\nb\nc", "jdd", "a\nc", Pos{1, 0}},
		{"2dd", "a\nb\nc", "2dd", "c", Pos{0, 0}},
		{"dd last", "a\nb", "jdd", "a", Pos{0, 0}},
		{"dd only", "a", "dd", "", Pos{0, 0}},
		{"dj", "a\nb\nc", "dj", "c", Pos{0, 0}},
		{"dG", "a\nb\nc", "jdG", "a", Pos{0, 0}},
		{"dgg", "a\nb\nc", "jdgg", "c", Pos{0, 0}},
		{"d$", "foo bar", "wd$", "foo ", Pos{0, 3}},
		{"D", "foo bar", "wD", "foo ", Pos{0, 3}},
		{"x count", "abcdef", "l3x", "aef", Pos{0, 1}},
		{"X", "abc", "$X", "ac", Pos{0, 1}},
		{"yy p", "a\nb", "yyp", "a\na\nb", Pos{1, 0}},
		{"yy P", "a\nb", "jyyP", "a\nb\nb", Pos{1, 0}},
		{"yw p", "ab cd", "ywP", "ab ab cd", Pos{0, 2}},
		{"cc", "  a\nb", "ccx<esc>", "  x\nb", Pos{0, 2}},
		{"cc only line", "a", "ccx<esc>", "x", Pos{0, 0}},
		{"o", "  a", "ox<esc>", "  a\n  x", Pos{1, 2}},
		{"O", "a", "Ox<esc>", "x\na", Pos{0, 0}},
		{"A", "ab", "Ac<esc>", "abc", Pos{0, 2}},
		{"I", "  ab", "$Ix<esc>", "  xab", Pos{0, 2}},
		{"a", "ab", "ax<esc>", "axb", Pos{0, 1}},
		{"insert enter indent", "  ab", "$i<enter>c<esc>", "  a\n  cb", Pos{1, 2}},
		{"insert backspace joins", "a\nb", "ji<backspace><esc>", "ab", Pos{0, 0}},
		{"u", "abc", "xu", "abc", Pos{0, 0}},
		{"u insert session", "abc", "ixyz<esc>u", "abc", Pos{0, 0}},
		{"redo", "abc", "xx2u<ctrl+r>", "bc", Pos{0, 0}},
		{"J", "a\n  b\nc", "J", "a b\nc", Pos{0, 1}},
		{"3J", "a\nb\nc", "3J", "a b c", Pos{0, 3}},
		{"r", "abc", "lrx", "axc", Pos{0, 1}},
		{"2r", "abc", "2rx", "xxc", Pos{0, 1}},
		{"~", "abC", "3~", "ABc", Pos{0, 2}},
		{"f", "a,b,c", "2f,D", "a,b", Pos{0, 2}},
		{"dt", "foo(bar)", "dt(", "(bar)", Pos{0, 0}},
		{"df", "foo(bar)", "df(", "bar)", Pos{0, 0}},
		{"dF", "foo(bar)", "$dF(", "foo)", Pos{0, 3}},
		{"diw", "foo bar baz", "wdiw", "foo  baz", Pos{0, 4}},
		{"daw", "foo bar baz", "wdaw", "foo baz", Pos{0, 4}},
		{`ci"`, `say "hello" now`, `fhci"x<esc>`, `say "x" now`, Pos{0, 5}},
		{`da"`, `a "b" c`, `fbda"`, "a c", Pos{0, 2}},
		{"di(", "f(a, (b))", "fadi(", "f()", Pos{0, 2}},
		{"ci{ multiline", "{\n  a\n}", "jci{x<esc>", "{\nx\n}", Pos{1, 0}},
		{"visual d", "abcdef", "lvlld", "aef", Pos{0, 1}},
		{"visual y P", "abc", "vlyP", "ababc", Pos{0, 1}},
		{"visual line d", "a\nb\nc", "Vjd", "c", Pos{0, 0}},
		{"visual c", "abc", "vlcx<esc>", "xc", Pos{0, 0}},
		{"visual o", "abcd", "lvlohd", "d", Pos{0, 0}},
		{"visual iw", "foo bar", "wviwd", "foo ", Pos{0, 3}},
		{"count G", "a\nb\nc", "2Gx", "a\n\nc", Pos{1, 0}},
		{"esc cancels", "abc", "d<esc>x", "bc", Pos{0, 0}},
		{"b e", "foo bar", "$bD", "foo ", Pos{0, 3}},
		{"e moves", "foo bar", "eD", "fo", Pos{0, 1}},
		{"c2w", "a b c", "c2wx<esc>", "x c", Pos{0, 0}},
		{"de", "foo bar", "de", " bar", Pos{0, 0}},
		{"paragraph", "a\n\nb\nc\n\nd", "}}dd", "a\n\nb\nc\nd", Pos{4, 0}},
		// the cursor stays on a character outside insert mode
		{"$x", "abc", "$x", "ab", Pos{0, 1}},
		{"A esc r", "ab", "AX<esc>rq", "abq", Pos{0, 2}},
		{"I esc a", "foo", "Iab<esc>az<esc>", "abzfoo", Pos{0, 2}},
		{"$ is sticky", "abc\nlonger line\nab", "$jjx", "abc\nlonger line\na", Pos{2, 0}},
		{"esc moves back", "abc", "lix<esc>", "axbc", Pos{0, 1}},
		{"esc at the start", "ab", "i<esc>x", "b", Pos{0, 0}},
		{"l stops on the last char", "ab", "5lx", "a", Pos{0, 0}},
		{"v$ takes the line break", "ab\ncd", "v$d", "cd", Pos{0, 0}},
		{"xp swaps", "abc", "xp", "bac", Pos{0, 1}},
		{"undo keeps the cursor on a char", "abc", "$xu", "abc", Pos{0, 2}},
		{"a at the end", "ab", "$ac<esc>", "abc", Pos{0, 2}},
		{"D then x", "abcd", "lDx", "", Pos{0, 0}},
		{"j onto a shorter line", "abcdef\nab", "$jx", "abcdef\na", Pos{1, 0}},
	} {
		e := New(c.text)
		feed(e, c.keys)
		if got := e.Text(); got != c.want {
			t.Errorf("%s: %q + %q = %q, want %q", c.name, c.text, c.keys, got, c.want)
			continue
		}
		if e.Cursor() != c.cur {
			t.Errorf("%s: cursor %v, want %v", c.name, e.Cursor(), c.cur)
		}
	}
}

func TestKeysLeftToTheApp(t *testing.T) {
	e := New("abc")
	for _, k := range []string{"s", "q", "?", "ctrl+n", "alt+1"} {
		if e.Key(k, "") {
			t.Errorf("normal %q used", k)
		}
	}
	e.Key("d", "d")
	if !e.Key("q", "q") || e.Pending() != "" {
		t.Error("a key completing nothing should cancel the pending command")
	}
	if e.Key("ctrl+c", "") {
		t.Error("ctrl+c with nothing pending belongs to the app")
	}
	e.Key("i", "i")
	if !e.Key("q", "q") || e.Key("ctrl+n", "") {
		t.Error("insert mode: q types, ctrl+n is the app's")
	}
	if e.Text() != "qabc" {
		t.Errorf("text = %q", e.Text())
	}
}

func TestPendingAndModes(t *testing.T) {
	e := New("abc")
	feed(e, "2d")
	if e.Pending() != "2d" {
		t.Fatalf("pending = %q", e.Pending())
	}
	feed(e, "<esc>v")
	if e.Mode() != Visual || e.Mode().String() != "VISUAL" {
		t.Fatalf("mode = %v", e.Mode())
	}
	a, b, ok := e.Selection()
	if !ok || a != (Pos{0, 0}) || b != (Pos{0, 1}) {
		t.Fatalf("selection = %v %v", a, b)
	}
}

func TestYankAndChanged(t *testing.T) {
	e := New("a\nb")
	feed(e, "yy")
	if s, ok := e.TakeYank(); !ok || s != "a" {
		t.Fatalf("yank = %q", s)
	}
	if e.Changed() {
		t.Fatal("a yank is no change")
	}
	feed(e, "dd")
	if !e.Changed() || e.Changed() {
		t.Fatal("Changed should report once")
	}
	e.Paste("x\ny")
	if e.Text() != "x\nyb" {
		t.Fatalf("paste = %q", e.Text())
	}
	feed(e, "u")
	if e.Text() != "b" {
		t.Fatalf("undo paste = %q", e.Text())
	}
}

func TestSetTextInVisualMode(t *testing.T) {
	e := New("a\nb\nc\nd")
	feed(e, "GVk")
	e.SetText("x")
	if e.Mode() != Normal {
		t.Fatalf("mode = %v", e.Mode())
	}
	if _, _, ok := e.Selection(); ok {
		t.Fatal("a selection over replaced text")
	}
	feed(e, "v")
	if a, b, ok := e.Selection(); !ok || a != (Pos{0, 0}) || b != (Pos{0, 1}) {
		t.Fatalf("selection = %v %v", a, b)
	}
}

func TestHugeCounts(t *testing.T) {
	e := New(strings.Repeat("x", 1000))
	feed(e, "yy999999p")
	if n := len(e.Text()); n > 2<<20 {
		t.Fatalf("a put of %d bytes", n)
	}
	feed(e, "u99999999999dd99999x")
	if e.LineCount() != 1 {
		t.Fatalf("lines = %d", e.LineCount())
	}
}

func TestReplace(t *testing.T) {
	e := New("hello world\nbye")
	feed(e, "wve")
	a, b, _ := e.Selection()
	end := e.Replace(a, b, "op{"+e.Text()[6:11]+"}")
	if e.Text() != "hello op{world}\nbye" || e.Mode() != Normal || e.Cursor() != (Pos{0, 6}) || end != (Pos{0, 15}) {
		t.Fatalf("got %q %v %v %v", e.Text(), e.Mode(), e.Cursor(), end)
	}
	if !e.Changed() {
		t.Error("not marked changed")
	}
	feed(e, "u")
	if e.Text() != "hello world\nbye" {
		t.Errorf("undo: %q", e.Text())
	}

	// Across lines, with a CRLF to normalize.
	e = New("ab\ncd")
	e.Replace(Pos{0, 1}, Pos{1, 1}, "X\r\nY")
	if e.Text() != "aX\nYd" {
		t.Errorf("got %q", e.Text())
	}
	// Out of range positions are clamped, and ends may come in any order.
	e = New("abc")
	e.Replace(Pos{5, 9}, Pos{0, 1}, "-")
	if e.Text() != "a-" {
		t.Errorf("got %q", e.Text())
	}
}

func TestReplaceInsert(t *testing.T) {
	e := New("x = ;")
	e.SetCursor(Pos{0, 4})
	e.ReplaceInsert(Pos{0, 4}, Pos{0, 4}, "op{", "}")
	if e.Mode() != Insert || e.Cursor() != (Pos{0, 7}) {
		t.Fatalf("got %v %v", e.Mode(), e.Cursor())
	}
	feed(e, "abc<esc>")
	if e.Text() != "x = op{abc};" {
		t.Fatalf("got %q", e.Text())
	}
	// The block and what was typed in it are one change.
	feed(e, "u")
	if e.Text() != "x = ;" {
		t.Errorf("undo: %q", e.Text())
	}

	// A visual selection is replaced, and the mode left.
	e = New("one two")
	feed(e, "v")
	a, b, _ := e.Selection()
	e.ReplaceInsert(a, b, "[", "]")
	feed(e, "<esc>")
	if e.Text() != "[]ne two" {
		t.Errorf("got %q", e.Text())
	}
}

func TestCountedInsert(t *testing.T) {
	for _, c := range []struct{ text, keys, want string }{
		{"ab", "3iX<esc>", "XXXab"},
		{"ab", "2aX<esc>", "aXXb"},
		{"ab", "2oX<esc>", "ab\nX\nX"},
		{"ab", "2OX<enter>Y<esc>", "X\nY\nX\nY\nab"},
		{"00000", "2O<tab><enter><esc>", "  \n  \n  \n  \n00000"},
	} {
		e := New(c.text)
		feed(e, c.keys)
		if e.Text() != c.want {
			t.Errorf("%q + %q = %q, want %q", c.text, c.keys, e.Text(), c.want)
		}
		if p := e.Cursor(); p.Line >= e.LineCount() || p.Col > len([]rune(e.Line(p.Line))) {
			t.Errorf("%q: cursor %v out of the text", c.keys, p)
		}
	}
}
