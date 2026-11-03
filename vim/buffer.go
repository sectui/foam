// Package vim is a vim-flavoured text editing engine: a buffer, its modes,
// motions, operators, text objects, registers and undo. It knows nothing of
// rendering: the app draws Lines, Cursor and Selection as it likes.
//
// As in vim, the cursor is on a character outside insert mode, and between
// two characters in insert mode: esc moves it back one.
package vim

import "strings"

type Pos struct{ Line, Col int }

func (a Pos) less(b Pos) bool { return a.Line < b.Line || (a.Line == b.Line && a.Col < b.Col) }

func order(a, b Pos) (Pos, Pos) {
	if b.less(a) {
		return b, a
	}
	return a, b
}

type buffer struct {
	lines [][]rune
}

func newBuffer(text string) buffer {
	parts := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	b := buffer{lines: make([][]rune, len(parts))}
	for i, p := range parts {
		b.lines[i] = []rune(p)
	}
	return b
}

func (b *buffer) text() string {
	parts := make([]string, len(b.lines))
	for i, l := range b.lines {
		parts[i] = string(l)
	}
	return strings.Join(parts, "\n")
}

func (b *buffer) clone() [][]rune {
	out := make([][]rune, len(b.lines))
	for i, l := range b.lines {
		out[i] = append([]rune(nil), l...)
	}
	return out
}

func (b *buffer) clamp(p Pos) Pos {
	p.Line = min(max(p.Line, 0), len(b.lines)-1)
	p.Col = min(max(p.Col, 0), len(b.lines[p.Line]))
	return p
}

func (b *buffer) get(a, c Pos) string {
	a, c = order(b.clamp(a), b.clamp(c))
	if a.Line == c.Line {
		return string(b.lines[a.Line][a.Col:c.Col])
	}
	var s strings.Builder
	s.WriteString(string(b.lines[a.Line][a.Col:]))
	for i := a.Line + 1; i < c.Line; i++ {
		s.WriteString("\n" + string(b.lines[i]))
	}
	s.WriteString("\n" + string(b.lines[c.Line][:c.Col]))
	return s.String()
}

func (b *buffer) remove(a, c Pos) string {
	a, c = order(b.clamp(a), b.clamp(c))
	s := b.get(a, c)
	head := append([]rune(nil), b.lines[a.Line][:a.Col]...)
	tail := b.lines[c.Line][c.Col:]
	b.lines[a.Line] = append(head, tail...)
	b.lines = append(b.lines[:a.Line+1], b.lines[c.Line+1:]...)
	return s
}

// insert puts s at p and returns the position after it.
func (b *buffer) insert(p Pos, s string) Pos {
	p = b.clamp(p)
	parts := strings.Split(s, "\n")
	line := b.lines[p.Line]
	tail := append([]rune(nil), line[p.Col:]...)
	first := append(append([]rune(nil), line[:p.Col]...), []rune(parts[0])...)
	if len(parts) == 1 {
		b.lines[p.Line] = append(first, tail...)
		return Pos{p.Line, len(first)}
	}
	newLines := make([][]rune, 0, len(parts))
	newLines = append(newLines, first)
	for _, mid := range parts[1 : len(parts)-1] {
		newLines = append(newLines, []rune(mid))
	}
	last := []rune(parts[len(parts)-1])
	end := Pos{p.Line + len(parts) - 1, len(last)}
	newLines = append(newLines, append(last, tail...))
	rest := append([][]rune(nil), b.lines[p.Line+1:]...)
	b.lines = append(append(b.lines[:p.Line], newLines...), rest...)
	return end
}

// removeLines deletes lines [from, to] and returns them, newline-joined.
func (b *buffer) removeLines(from, to int) string {
	from, to = max(from, 0), min(to, len(b.lines)-1)
	parts := make([]string, 0, to-from+1)
	for i := from; i <= to; i++ {
		parts = append(parts, string(b.lines[i]))
	}
	b.lines = append(b.lines[:from], b.lines[to+1:]...)
	if len(b.lines) == 0 {
		b.lines = [][]rune{{}}
	}
	return strings.Join(parts, "\n")
}

func (b *buffer) insertLines(at int, s string) {
	parts := strings.Split(s, "\n")
	ins := make([][]rune, len(parts))
	for i, p := range parts {
		ins[i] = []rune(p)
	}
	at = min(max(at, 0), len(b.lines))
	rest := append([][]rune(nil), b.lines[at:]...)
	b.lines = append(append(b.lines[:at], ins...), rest...)
}

func firstNonBlank(l []rune) int {
	for i, r := range l {
		if r != ' ' && r != '\t' {
			return i
		}
	}
	return len(l)
}
