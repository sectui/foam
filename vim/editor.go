package vim

import (
	"strings"
	"unicode"
)

type Mode int

const (
	Normal Mode = iota
	Insert
	Visual
	VisualLine
)

func (m Mode) String() string {
	switch m {
	case Insert:
		return "INSERT"
	case Visual:
		return "VISUAL"
	case VisualLine:
		return "V-LINE"
	}
	return "NORMAL"
}

type register struct {
	text     string
	linewise bool
}

type snapshot struct {
	lines [][]rune
	cur   Pos
}

type motionKind int

const (
	exclusive motionKind = iota
	inclusive
	linewise
)

// Editor is the engine. Feed it keys with Key, read it with Lines, Cursor,
// Selection and Mode.
type Editor struct {
	buf    buffer
	cur    Pos
	want   int // column kept across vertical moves
	mode   Mode
	anchor Pos

	count1, count2 string
	op             rune
	prefix         string // g, r, f, t, F, T, i, a (text object)

	// 3iX inserts XXX: the count and where the insert started
	repeat      int
	repeatLines bool
	insertStart Pos

	reg        register
	yanked     *string
	undo, redo []snapshot
	changed    bool
	version    uint64

	TabText string

	// ReadOnly refuses the keys that would change the text: the motions, the
	// counts, the visual modes and the yanks are left. SetText still works,
	// for an app that owns the text.
	ReadOnly bool
}

func New(text string) *Editor {
	return &Editor{buf: newBuffer(text), TabText: "  "}
}

func (e *Editor) Text() string      { return e.buf.text() }
func (e *Editor) Mode() Mode        { return e.mode }
func (e *Editor) Cursor() Pos       { return e.cur }
func (e *Editor) LineCount() int    { return len(e.buf.lines) }
func (e *Editor) Line(i int) string { return string(e.buf.lines[i]) }

func (e *Editor) Lines() []string {
	out := make([]string, len(e.buf.lines))
	for i, l := range e.buf.lines {
		out[i] = string(l)
	}
	return out
}

// Pending is the command typed so far, like "2d" or "g".
func (e *Editor) Pending() string {
	s := e.count1
	if e.op != 0 {
		s += string(e.op)
	}
	return s + e.count2 + e.prefix
}

func (e *Editor) touch() {
	e.changed = true
	e.version++
}

// Version changes whenever the text does: a cheap way to know whether
// something derived from the text is stale.
func (e *Editor) Version() uint64 { return e.version }

// Changed reports whether the text changed since the last call.
func (e *Editor) Changed() bool {
	c := e.changed
	e.changed = false
	return c
}

// TakeYank returns what the last yank or delete put in the register, once.
func (e *Editor) TakeYank() (string, bool) {
	if e.yanked == nil {
		return "", false
	}
	s := *e.yanked
	e.yanked = nil
	return s, true
}

// SetText replaces the whole text as one undoable change, keeping the cursor
// where it can.
func (e *Editor) SetText(text string) {
	if text == e.Text() {
		return
	}
	e.snap()
	e.buf = newBuffer(text)
	e.cur = e.buf.clamp(e.cur)
	e.anchor = e.buf.clamp(e.anchor)
	if e.mode == Visual || e.mode == VisualLine {
		e.mode = Normal // what was selected is gone
	}
	e.reset()
	e.settle()
	e.touch()
}

func (e *Editor) SetCursor(p Pos) {
	e.cur = e.buf.clamp(p)
	e.settle()
	e.want = e.cur.Col
}

// Selection is the visual selection, [start, end), when there is one.
func (e *Editor) Selection() (start, end Pos, ok bool) {
	e.anchor, e.cur = e.buf.clamp(e.anchor), e.buf.clamp(e.cur)
	switch e.mode {
	case Visual:
		a, b := order(e.anchor, e.cur)
		if b.Col >= len(e.buf.lines[b.Line]) && b.Line < len(e.buf.lines)-1 {
			return a, Pos{b.Line + 1, 0}, true
		}
		return a, Pos{b.Line, min(b.Col+1, len(e.buf.lines[b.Line]))}, true
	case VisualLine:
		a, b := order(e.anchor, e.cur)
		return Pos{a.Line, 0}, Pos{b.Line, len(e.buf.lines[b.Line])}, true
	}
	return Pos{}, Pos{}, false
}

// Replace swaps [a, b) for s as one undoable change, and returns the end
// of s. It goes back to normal mode, the cursor at a.
func (e *Editor) Replace(a, b Pos, s string) Pos {
	e.snap()
	end := e.replace(a, b, s)
	e.mode = Normal
	e.cur = e.onChar(a)
	e.want = e.cur.Col
	return end
}

// ReplaceInsert swaps [a, b) for before+after and enters insert mode between
// them: what is typed next belongs to the same undoable change.
func (e *Editor) ReplaceInsert(a, b Pos, before, after string) {
	e.snap()
	a, _ = order(e.buf.clamp(a), e.buf.clamp(b))
	e.replace(a, b, after)
	e.cur = e.buf.insert(a, strings.ReplaceAll(before, "\r\n", "\n"))
	e.want = e.cur.Col
	e.mode = Insert
}

func (e *Editor) replace(a, b Pos, s string) Pos {
	a, b = order(e.buf.clamp(a), e.buf.clamp(b))
	e.buf.remove(a, b)
	end := e.buf.insert(a, strings.ReplaceAll(s, "\r\n", "\n"))
	e.reset()
	e.touch()
	return end
}

func (e *Editor) snap() {
	e.undo = append(e.undo, snapshot{e.buf.clone(), e.cur})
	if len(e.undo) > 500 {
		e.undo = e.undo[1:]
	}
	e.redo = nil
}

func (e *Editor) restore(from, to *[]snapshot) {
	if len(*from) == 0 {
		return
	}
	s := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	*to = append(*to, snapshot{e.buf.clone(), e.cur})
	e.buf.lines = s.lines
	e.cur = e.onChar(s.cur)
	e.touch()
}

func (e *Editor) setReg(text string, lw bool) {
	e.reg = register{text, lw}
	e.yanked = &text
}

func (e *Editor) reset() {
	e.count1, e.count2, e.op, e.prefix = "", "", 0, ""
}

func (e *Editor) count() int {
	n := 1
	if e.count1 != "" {
		n = atoi(e.count1)
	}
	if e.count2 != "" {
		n *= atoi(e.count2)
	}
	return min(max(n, 1), 1_000_000)
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
		if n > 1_000_000 {
			return 1_000_000
		}
	}
	return n
}

// Paste inserts s at the cursor, as one change. It does nothing while ReadOnly.
func (e *Editor) Paste(s string) {
	if e.ReadOnly {
		return
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if e.mode != Insert {
		e.snap()
	}
	if e.mode == Visual || e.mode == VisualLine {
		a, b, _ := e.Selection()
		e.buf.remove(a, b)
		e.cur = a
		e.mode = Normal
	}
	e.cur = e.buf.insert(e.cur, s)
	e.want = e.cur.Col
	e.touch()
}

// Key handles one key, named as bubbletea names them ("ctrl+c", "enter",
// "x"); text is what it types, "" for a non-printing key. It reports whether
// the key was used: an app gives the others its own meaning.
func (e *Editor) Key(name, text string) bool {
	if e.ReadOnly && e.mutates(name) {
		return false
	}
	var used bool
	switch e.mode {
	case Insert:
		used = e.insertKey(name, text)
	case Visual, VisualLine:
		used = e.visualKey(name, text)
	default:
		used = e.normalKey(name, text)
	}
	e.settle()
	return used
}

// mutates reports whether the key would change the text in the current mode.
// Insert mode is unreachable while ReadOnly: every key opening it is refused,
// here and in visual mode (c, s), so nothing else needs checking.
func (e *Editor) mutates(name string) bool {
	switch e.mode {
	case Insert:
		return true
	case Visual, VisualLine:
		switch name {
		case "d", "c", "s", "x", "delete", "p", "P", "~":
			return true
		}
		return false
	}
	switch name {
	case "d", "c", "s", "S", "x", "X", "delete", "D", "C", "R", "p", "P", "J", "~", "u", "ctrl+r", "o", "O":
		return true
	case "r":
		// replaces a character, unless an operator is waiting: then it is dropped
		return e.op == 0
	case "i", "a", "I", "A":
		return e.op == 0
	}
	return false
}

// endOfLine is the column wanted after $: the end of every line moved to.
const endOfLine = 1 << 30

// settle puts the cursor on a character in normal mode, as vim does. In
// visual mode it may also be on the line break, to select it.
func (e *Editor) settle() {
	if e.mode == Normal {
		e.cur = e.onChar(e.cur)
	}
}

func (e *Editor) onChar(p Pos) Pos {
	p = e.buf.clamp(p)
	p.Col = min(p.Col, max(len(e.buf.lines[p.Line])-1, 0))
	return p
}

func (e *Editor) insertKey(name, text string) bool {
	switch name {
	case "esc", "ctrl+c", "ctrl+[":
		e.mode = Normal
		e.repeatInsert()
		if e.cur.Col > 0 {
			e.cur.Col--
		}
	case "enter":
		indent := string(e.buf.lines[e.cur.Line][:firstNonBlank(e.buf.lines[e.cur.Line])])
		indent = indent[:min(len(indent), len(string(e.buf.lines[e.cur.Line][:e.cur.Col])))]
		e.cur = e.buf.insert(e.cur, "\n"+indent)
	case "tab":
		e.cur = e.buf.insert(e.cur, e.TabText)
	case "backspace", "shift+backspace":
		if p, ok := e.buf.prev(e.cur); ok {
			e.buf.remove(p, e.cur)
			e.cur = p
		}
	case "delete":
		if p, ok := e.buf.next(e.cur); ok {
			e.buf.remove(e.cur, p)
		}
	case "ctrl+w", "alt+backspace":
		p := e.buf.wordBackward(e.cur, false)
		e.buf.remove(p, e.cur)
		e.cur = p
	case "ctrl+u":
		p := Pos{e.cur.Line, 0}
		e.buf.remove(p, e.cur)
		e.cur = p
	case "left":
		e.cur.Col = max(e.cur.Col-1, 0)
	case "right":
		e.cur.Col = min(e.cur.Col+1, len(e.buf.lines[e.cur.Line]))
	case "up", "down":
		d := 1
		if name == "up" {
			d = -1
		}
		e.vertical(d)
		return true
	case "home":
		e.cur.Col = 0
	case "end":
		e.cur.Col = len(e.buf.lines[e.cur.Line])
	default:
		if text == "" || strings.HasPrefix(name, "ctrl+") || strings.HasPrefix(name, "alt+") {
			return false
		}
		e.cur = e.buf.insert(e.cur, text)
	}
	if name != "esc" && name != "ctrl+c" && name != "ctrl+[" && !strings.HasPrefix(name, "left") &&
		name != "right" && name != "home" && name != "end" {
		e.touch()
	}
	e.want = e.cur.Col
	return true
}

func (e *Editor) vertical(d int) {
	e.cur.Line = min(max(e.cur.Line+d, 0), len(e.buf.lines)-1)
	e.cur.Col = min(e.want, len(e.buf.lines[e.cur.Line]))
}

func (e *Editor) motion(name, text string, n int) (Pos, motionKind, bool) {
	b := &e.buf
	p := e.cur
	switch name {
	case "h", "left":
		return Pos{p.Line, max(p.Col-n, 0)}, exclusive, true
	case "l", "right", "space":
		return Pos{p.Line, min(p.Col+n, len(b.lines[p.Line]))}, exclusive, true
	case "j", "down":
		l := min(p.Line+n, len(b.lines)-1)
		return Pos{l, min(e.want, len(b.lines[l]))}, linewise, true
	case "k", "up":
		l := max(p.Line-n, 0)
		return Pos{l, min(e.want, len(b.lines[l]))}, linewise, true
	case "w", "W":
		for range n {
			q := b.wordForward(p, name == "W")
			if q == p {
				break
			}
			p = q
		}
		return p, exclusive, true
	case "b", "B":
		for range n {
			q := b.wordBackward(p, name == "B")
			if q == p {
				break
			}
			p = q
		}
		return p, exclusive, true
	case "e", "E":
		for range n {
			q := b.wordEnd(p, name == "E")
			if q == p {
				break
			}
			p = q
		}
		return p, inclusive, true
	case "0", "home":
		return Pos{p.Line, 0}, exclusive, true
	case "^":
		return Pos{p.Line, firstNonBlank(b.lines[p.Line])}, exclusive, true
	case "$", "end":
		if n > 1 && p.Line == len(b.lines)-1 {
			return p, exclusive, true // a count it cannot make: no move
		}
		l := min(p.Line+n-1, len(b.lines)-1)
		return Pos{l, len(b.lines[l])}, exclusive, true
	case "G":
		l := len(b.lines) - 1
		if e.count1 != "" || e.count2 != "" {
			l = min(n, len(b.lines)) - 1
		}
		return Pos{l, firstNonBlank(b.lines[l])}, linewise, true
	case "}":
		l := p.Line
		for i := range n {
			if l >= len(b.lines)-1 {
				if i > 0 {
					return p, exclusive, true // a count it cannot make: no move
				}
				break
			}
			l++
			for l < len(b.lines)-1 && len(strings.TrimSpace(string(b.lines[l]))) > 0 {
				l++
			}
		}
		if l >= len(b.lines)-1 {
			l = len(b.lines) - 1
			return Pos{l, len(b.lines[l])}, exclusive, true
		}
		return Pos{l, 0}, exclusive, true
	case "{":
		l := p.Line
		for i := range n {
			if l <= 0 {
				if i > 0 {
					return p, exclusive, true
				}
				break
			}
			l--
			for l > 0 && len(strings.TrimSpace(string(b.lines[l]))) > 0 {
				l--
			}
		}
		return Pos{max(l, 0), 0}, exclusive, true
	}
	return p, exclusive, false
}

func (e *Editor) findChar(kind string, c rune, n int) (Pos, motionKind, bool) {
	l := e.buf.lines[e.cur.Line]
	col := e.cur.Col
	switch kind {
	case "f", "t":
		start := col + 1
		if kind == "t" {
			start++
		}
		for i := start - 1; i < len(l); i++ {
			if i >= 0 && i > col && l[i] == c {
				if n--; n == 0 {
					if kind == "t" {
						i--
					}
					return Pos{e.cur.Line, i}, inclusive, true
				}
			}
		}
	case "F", "T":
		for i := col - 1; i >= 0; i-- {
			if l[i] == c {
				if n--; n == 0 {
					if kind == "T" {
						i++
					}
					return Pos{e.cur.Line, i}, exclusive, true
				}
			}
		}
	}
	return e.cur, exclusive, false
}

func (e *Editor) normalKey(name, text string) bool {
	pending := e.Pending() != ""
	if name == "esc" || name == "ctrl+c" || name == "ctrl+[" {
		e.reset()
		return pending
	}

	// Digits make a count, except a leading 0, which is a motion.
	if len(name) == 1 && name[0] >= '0' && name[0] <= '9' && e.prefix == "" && (name != "0" || e.count2 != "" || (e.op == 0 && e.count1 != "")) {
		if e.op == 0 {
			e.count1 += name
		} else {
			e.count2 += name
		}
		return true
	}

	n := e.count()
	if e.prefix != "" {
		prefix := e.prefix
		e.prefix = ""
		r := []rune(text)
		switch {
		case prefix == "g" && name == "g":
			p := Pos{0, 0}
			if e.count1 != "" || e.count2 != "" {
				p.Line = min(n, len(e.buf.lines)) - 1
			}
			p.Col = firstNonBlank(e.buf.lines[p.Line])
			e.apply(p, linewise, "gg")
		case prefix == "r" && len(r) == 1:
			e.replaceChar(r[0], n)
		case strings.Contains("ftFT", prefix) && len(r) == 1:
			if p, kind, ok := e.findChar(prefix, r[0], n); ok {
				e.apply(p, kind, prefix)
			} else {
				e.reset()
			}
		case (prefix == "i" || prefix == "a") && len(r) == 1:
			if a, b, ok := e.buf.textObject(e.cur, prefix == "a", r[0]); ok {
				if strings.ContainsRune("\"'`", r[0]) && prefix == "i" && n > 1 {
					// 2i" takes the quotes too
					a.Col, b.Col = a.Col-1, b.Col+1
				}
				for i := 1; i < n && ok && (r[0] == 'w' || r[0] == 'W'); i++ {
					b, ok = e.buf.nextWordObject(b, prefix == "a", r[0] == 'W')
				}
				if ok {
					e.operate(a, b)
				}
			}
			e.reset()
		default:
			e.reset()
		}
		return true
	}

	if p, kind, ok := e.motion(name, text, n); ok {
		e.apply(p, kind, name)
		return true
	}

	switch name {
	case "g", "f", "t", "F", "T":
		e.prefix = name
		return true
	case "i", "a", "r":
		if (e.op != 0) != (name == "r") {
			e.prefix = name
			return true
		}
	case "d", "c", "y":
		r := rune(name[0])
		switch e.op {
		case 0:
			e.op = r
		case r:
			e.lineOp(n)
		default:
			e.reset()
		}
		return true
	}
	if e.op != 0 {
		e.reset()
		return true
	}

	defer e.reset()
	l := e.buf.lines[e.cur.Line]
	if strings.Contains("iaIAoO", name) && len(name) == 1 {
		e.repeat, e.repeatLines = n, name == "o" || name == "O"
	}
	switch name {
	case "i":
		e.startInsert(e.cur)
	case "a":
		e.startInsert(Pos{e.cur.Line, min(e.cur.Col+1, len(l))})
	case "I":
		e.startInsert(Pos{e.cur.Line, firstNonBlank(l)})
	case "A":
		e.startInsert(Pos{e.cur.Line, len(l)})
	case "o", "O":
		e.snap()
		at := e.cur.Line + 1
		if name == "O" {
			at = e.cur.Line
		}
		indent := string(l[:firstNonBlank(l)])
		e.buf.insertLines(at, indent)
		e.cur = Pos{at, len([]rune(indent))}
		e.insertStart = Pos{at, 0}
		e.mode = Insert
		e.touch()
	case "x", "delete":
		if len(l) > 0 {
			e.snap()
			end := min(e.cur.Col+n, len(l))
			e.setReg(e.buf.remove(e.cur, Pos{e.cur.Line, end}), false)
			e.touch()
		}
	case "X":
		if e.cur.Col > 0 {
			e.snap()
			start := max(e.cur.Col-n, 0)
			e.setReg(e.buf.remove(Pos{e.cur.Line, start}, e.cur), false)
			e.cur.Col = start
			e.touch()
		}
	case "D", "C":
		if n > 1 && e.cur.Line == len(e.buf.lines)-1 {
			return true
		}
		last := min(e.cur.Line+n-1, len(e.buf.lines)-1)
		if name == "D" && last > e.cur.Line && e.cur.Col <= firstNonBlank(l) {
			e.op = 'd'
			e.lines(e.cur.Line, last)
			return true
		}
		e.snap()
		e.setReg(e.buf.remove(e.cur, Pos{last, len(e.buf.lines[last])}), false)
		e.touch()
		if name == "C" {
			e.mode = Insert
		}
	case "Y":
		e.op = 'y'
		e.lineOp(n)
	case "p", "P":
		e.put(name == "P", n)
	case "J":
		e.join(max(n, 2) - 1)
	case "~":
		e.toggleCase(n)
	case "u":
		for range n {
			e.restore(&e.undo, &e.redo)
		}
	case "ctrl+r":
		for range n {
			e.restore(&e.redo, &e.undo)
		}
	case "v":
		e.mode, e.anchor = Visual, e.cur
		if n > 1 {
			e.cur.Col = min(e.cur.Col+n-1, len(l))
		}
	case "V":
		e.mode, e.anchor = VisualLine, e.cur
	default:
		return pending
	}
	return true
}

func (e *Editor) startInsert(p Pos) {
	e.snap()
	e.cur = e.buf.clamp(p)
	e.mode = Insert
	e.insertStart = e.cur
}

func (e *Editor) repeatInsert() {
	n, lines := e.repeat, e.repeatLines
	e.repeat = 0
	if n < 2 || e.cur.less(e.insertStart) {
		return
	}
	typed := e.buf.get(e.insertStart, e.cur)
	if typed == "" || strings.Count(typed, "\n") > 0 && !lines {
		return
	}
	n = min(n, maxPut/(len(typed)+1))
	if lines {
		block := strings.Repeat("\n"+typed, n-1)[1:]
		e.buf.insertLines(e.cur.Line+1, block)
		e.cur = Pos{e.cur.Line + (n-1)*(strings.Count(typed, "\n")+1), len([]rune(typed[strings.LastIndexByte(typed, '\n')+1:]))}
	} else {
		e.cur = e.buf.insert(e.cur, strings.Repeat(typed, n-1))
	}
	e.cur = e.buf.clamp(e.cur)
	e.touch()
}

// apply moves to p, or applies the pending operator up to it.
func (e *Editor) apply(p Pos, kind motionKind, name string) {
	if e.op == 0 {
		e.cur = e.buf.clamp(p)
		if kind != linewise || name == "G" || name == "gg" {
			e.want = e.cur.Col
		}
		if name == "$" || name == "end" {
			e.want = endOfLine
		}
		e.reset()
		return
	}
	start := e.cur
	// A motion that cannot move cancels the operator, as in vim.
	vertical := strings.Contains(" j k down up ", " "+name+" ")
	failed := p == start && (strings.Contains(" b B { } ", " "+name+" ") || (name == "$" || name == "end") && e.count() > 1)
	if failed || vertical && p.Line == start.Line {
		e.reset()
		return
	}
	last := len(e.buf.lines) - 1
	if (name == "$" || name == "end") && p.Line > start.Line && start.Col <= firstNonBlank(e.buf.lines[start.Line]) {
		// 3d$ from the start of a line works on whole lines, as 3D
		kind = linewise
	}
	if kind == exclusive && p == (Pos{last, len(e.buf.lines[last])}) && p.Line > start.Line && start.Col <= firstNonBlank(e.buf.lines[start.Line]) {
		// ending the text from the start of a line, as if at column 0 below
		kind = linewise
	}
	if kind == linewise {
		a, b := order(start, p)
		e.lines(a.Line, b.Line)
		return
	}
	word := name == "w" || name == "W"
	switch {
	case word && e.op == 'c' && e.buf.classAt(start, name == "W") != space:
		// cw changes to the end of the word, as ce does.
		p = e.runEnd(start, name == "W")
		for range e.count() - 1 {
			q := e.buf.wordEnd(Pos{p.Line, max(p.Col-1, 0)}, name == "W")
			q.Col = min(q.Col+1, len(e.buf.lines[q.Line]))
			if q == p {
				break
			}
			p = q
		}
	case word:
		// The last word moved over ends its line: the operator stops there.
		q, prev := start, start
		for range e.count() {
			next := e.buf.wordForward(q, name == "W")
			if next == q {
				break
			}
			prev, q = q, next
		}
		if q.Line > prev.Line {
			if prev.Line > start.Line && start.Col <= firstNonBlank(e.buf.lines[start.Line]) {
				// 2dw over whole lines from their start
				e.lines(start.Line, prev.Line)
				return
			}
			q = Pos{prev.Line, len(e.buf.lines[prev.Line])}
		}
		p = q
	case kind == inclusive:
		p = Pos{p.Line, min(p.Col+1, len(e.buf.lines[p.Line]))}
	}
	if a, b := order(start, p); kind == exclusive && b.Col == 0 && b.Line > a.Line {
		// An exclusive motion ending at column 0 ends with the line before,
		// and works on whole lines if it starts before the first non-blank.
		if a.Col <= firstNonBlank(e.buf.lines[a.Line]) {
			e.lines(a.Line, b.Line-1)
			return
		}
		end := Pos{b.Line - 1, len(e.buf.lines[b.Line-1])}
		if p == b {
			p = end
		} else {
			start = end
		}
	}
	e.operate(start, p)
	e.reset()
}

func (e *Editor) runEnd(p Pos, big bool) Pos {
	l := e.buf.lines[p.Line]
	c := e.buf.classAt(p, big)
	i := p.Col
	for i < len(l) && classOf(l[i], big) == c {
		i++
	}
	return Pos{p.Line, i}
}

func (e *Editor) operate(a, b Pos) {
	a, b = order(a, b)
	switch e.op {
	case 'y':
		e.setReg(e.buf.get(a, b), false)
		e.cur = a
	case 'd':
		e.snap()
		e.setReg(e.buf.remove(a, b), false)
		e.cur = e.buf.clamp(a)
		e.touch()
	case 'c':
		e.snap()
		e.setReg(e.buf.remove(a, b), false)
		e.cur = e.buf.clamp(a)
		e.mode = Insert
		e.touch()
	}
	e.want = e.cur.Col
}

func (e *Editor) lineOp(n int) {
	if n > 1 && e.cur.Line == len(e.buf.lines)-1 {
		e.reset()
		return
	}
	e.lines(e.cur.Line, min(e.cur.Line+n-1, len(e.buf.lines)-1))
}

func (e *Editor) lines(from, to int) {
	defer e.reset()
	switch e.op {
	case 'y':
		e.setReg(e.buf.get(Pos{from, 0}, Pos{to, len(e.buf.lines[to])}), true)
		e.cur = Pos{from, e.cur.Col}
		if from != e.cur.Line {
			e.cur.Col = 0
		}
		e.cur = e.buf.clamp(e.cur)
	case 'd':
		e.snap()
		e.setReg(e.buf.removeLines(from, to), true)
		l := min(from, len(e.buf.lines)-1)
		e.cur = Pos{l, firstNonBlank(e.buf.lines[l])}
		e.touch()
	case 'c':
		e.snap()
		indent := string(e.buf.lines[from][:firstNonBlank(e.buf.lines[from])])
		e.setReg(e.replaceLines(from, to, indent), true)
		e.cur = Pos{from, len([]rune(indent))}
		e.mode = Insert
		e.touch()
	}
	e.want = e.cur.Col
}

func (e *Editor) replaceLines(from, to int, with string) string {
	all := to-from+1 >= len(e.buf.lines)
	removed := e.buf.removeLines(from, to)
	if all {
		e.buf.lines = [][]rune{[]rune(with)}
	} else {
		e.buf.insertLines(from, with)
	}
	return removed
}

// maxPut bounds what one put may insert: 999999p must not take the memory.
const maxPut = 1 << 20

func (e *Editor) put(before bool, n int) {
	if e.reg.text == "" && !e.reg.linewise {
		return
	}
	n = max(min(n, maxPut/max(len(e.reg.text)+1, 1)), 1)
	e.snap()
	text := strings.Repeat(e.reg.text+"\n", n)
	text = text[:len(text)-1]
	if e.reg.linewise {
		at := e.cur.Line + 1
		if before {
			at = e.cur.Line
		}
		e.buf.insertLines(at, text)
		e.cur = Pos{at, firstNonBlank(e.buf.lines[at])}
	} else {
		text = strings.Repeat(e.reg.text, n)
		p := e.cur
		if !before {
			p.Col = min(p.Col+1, len(e.buf.lines[p.Line]))
		}
		end := e.buf.insert(p, text)
		e.cur, _ = e.buf.prev(end)
		if end.Col == 0 {
			e.cur = end
		}
	}
	e.want = e.cur.Col
	e.touch()
}

func (e *Editor) join(n int) {
	if e.cur.Line+1 >= len(e.buf.lines) {
		return
	}
	e.snap()
	for range n {
		if e.cur.Line+1 >= len(e.buf.lines) {
			break
		}
		l := e.buf.lines[e.cur.Line]
		next := e.buf.lines[e.cur.Line+1]
		trimmed := []rune(strings.TrimLeft(string(next), " \t"))
		end := Pos{e.cur.Line, len(l)}
		e.buf.remove(end, Pos{e.cur.Line + 1, len(next) - len(trimmed)})
		if len(trimmed) > 0 && len(l) > 0 && l[len(l)-1] != ' ' {
			e.buf.insert(end, " ")
		}
		e.cur = end
	}
	e.touch()
}

func (e *Editor) replaceChar(r rune, n int) {
	l := e.buf.lines[e.cur.Line]
	if e.cur.Col+n > len(l) {
		e.reset()
		return
	}
	e.snap()
	for i := range n {
		l[e.cur.Col+i] = r
	}
	e.cur.Col += n - 1
	e.touch()
	e.reset()
}

func (e *Editor) toggleCase(n int) {
	l := e.buf.lines[e.cur.Line]
	if e.cur.Col >= len(l) {
		return
	}
	e.snap()
	end := min(e.cur.Col+n, len(l))
	for i := e.cur.Col; i < end; i++ {
		if unicode.IsUpper(l[i]) {
			l[i] = unicode.ToLower(l[i])
		} else {
			l[i] = unicode.ToUpper(l[i])
		}
	}
	e.cur.Col = min(end, len(l))
	e.touch()
}

func (e *Editor) visualKey(name, text string) bool {
	n := e.count()
	if e.prefix != "" {
		prefix := e.prefix
		e.prefix = ""
		r := []rune(text)
		switch {
		case prefix == "g" && name == "g":
			e.cur = Pos{0, 0}
		case strings.Contains("ftFT", prefix) && len(r) == 1:
			if p, _, ok := e.findChar(prefix, r[0], n); ok {
				e.cur = p
			}
		case (prefix == "i" || prefix == "a") && len(r) == 1:
			if a, b, ok := e.buf.textObject(e.cur, prefix == "a", r[0]); ok && b != a {
				e.anchor = a
				e.cur, _ = e.buf.prev(b)
			}
		}
		e.count1 = ""
		return true
	}
	if len(name) == 1 && name[0] >= '0' && name[0] <= '9' && (name != "0" || e.count1 != "") {
		e.count1 += name
		return true
	}
	if p, _, ok := e.motion(name, text, n); ok {
		e.cur = e.buf.clamp(p)
		switch name {
		case "j", "k", "up", "down":
		case "$", "end":
			e.want = endOfLine
		default:
			e.want = e.cur.Col
		}
		e.count1 = ""
		return true
	}
	defer func() { e.count1 = "" }()
	switch name {
	case "esc", "ctrl+c", "ctrl+[":
		e.mode = Normal
	case "v":
		if e.mode == Visual {
			e.mode = Normal
		} else {
			e.mode = Visual
		}
	case "V":
		if e.mode == VisualLine {
			e.mode = Normal
		} else {
			e.mode = VisualLine
		}
	case "o":
		e.anchor, e.cur = e.cur, e.anchor
	case "g", "f", "t", "F", "T", "i", "a":
		e.prefix = name
	case "d", "x", "delete", "y", "c", "s":
		a, b, _ := e.Selection()
		lw := e.mode == VisualLine
		switch {
		case name == "y":
			e.setReg(e.buf.get(a, b), lw)
			e.cur = a
		case lw:
			e.snap()
			if name == "c" || name == "s" {
				e.setReg(e.replaceLines(a.Line, b.Line, ""), true)
			} else {
				e.setReg(e.buf.removeLines(a.Line, b.Line), true)
			}
			l := min(a.Line, len(e.buf.lines)-1)
			e.cur = Pos{l, 0}
			e.touch()
		default:
			e.snap()
			e.setReg(e.buf.remove(a, b), false)
			e.cur = e.buf.clamp(a)
			e.touch()
		}
		e.mode = Normal
		if name == "c" || name == "s" {
			e.mode = Insert
		}
	case "p", "P":
		a, b, _ := e.Selection()
		reg := e.reg
		lw := e.mode == VisualLine
		e.snap()
		var removed string
		switch {
		case lw && reg.text == "" && !reg.linewise:
			removed = e.buf.removeLines(a.Line, b.Line)
			l := min(a.Line, len(e.buf.lines)-1)
			e.cur = Pos{l, firstNonBlank(e.buf.lines[l])}
		case lw:
			// what is put over lines goes on lines of its own
			removed = e.replaceLines(a.Line, b.Line, reg.text)
			e.cur = Pos{a.Line, firstNonBlank(e.buf.lines[a.Line])}
		case reg.linewise:
			removed = e.buf.remove(a, b)
			e.buf.insert(a, "\n"+reg.text+"\n")
			e.cur = Pos{a.Line + 1, firstNonBlank(e.buf.lines[a.Line+1])}
		default:
			removed = e.buf.remove(a, b)
			end := e.buf.insert(a, reg.text)
			e.cur, _ = e.buf.prev(end)
			if reg.text == "" {
				e.cur = a
			}
		}
		e.setReg(removed, lw)
		e.mode = Normal
		e.touch()
	case "~":
		a, b, _ := e.Selection()
		e.snap()
		s := []rune(e.buf.remove(a, b))
		for i, r := range s {
			if unicode.IsUpper(r) {
				s[i] = unicode.ToLower(r)
			} else {
				s[i] = unicode.ToUpper(r)
			}
		}
		e.buf.insert(a, string(s))
		e.cur = a
		e.mode = Normal
		e.touch()
	default:
		// In visual mode, a printable key never reaches the app.
		return text != "" && !strings.HasPrefix(name, "ctrl+") && !strings.HasPrefix(name, "alt+")
	}
	return true
}
