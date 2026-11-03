package vim

import "unicode"

type class int

const (
	space class = iota
	word
	punctuation
)

func classOf(r rune, big bool) class {
	switch {
	case unicode.IsSpace(r):
		return space
	case big || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
		return word
	}
	return punctuation
}

// at returns the rune at p, '\n' at the end of a line.
func (b *buffer) at(p Pos) rune {
	if p.Col >= len(b.lines[p.Line]) {
		return '\n'
	}
	return b.lines[p.Line][p.Col]
}

// next steps one character forward, through line ends; ok is false at the end.
func (b *buffer) next(p Pos) (Pos, bool) {
	if p.Col < len(b.lines[p.Line]) {
		return Pos{p.Line, p.Col + 1}, true
	}
	if p.Line+1 < len(b.lines) {
		return Pos{p.Line + 1, 0}, true
	}
	return p, false
}

func (b *buffer) prev(p Pos) (Pos, bool) {
	if p.Col > 0 {
		return Pos{p.Line, p.Col - 1}, true
	}
	if p.Line > 0 {
		return Pos{p.Line - 1, len(b.lines[p.Line-1])}, true
	}
	return p, false
}

func (b *buffer) classAt(p Pos, big bool) class {
	r := b.at(p)
	if r == '\n' {
		return space
	}
	return classOf(r, big)
}

// wordForward is w: the start of the next word; an empty line counts as one.
func (b *buffer) wordForward(p Pos, big bool) Pos {
	start := b.classAt(p, big)
	q, ok := p, true
	if start != space {
		for ok && b.classAt(q, big) == start && q.Line == p.Line {
			q, ok = b.next(q)
		}
	}
	for ok && b.classAt(q, big) == space {
		if q.Line != p.Line && len(b.lines[q.Line]) == 0 {
			return q
		}
		q, ok = b.next(q)
	}
	return q
}

// wordEnd is e: the last character of the current or next word.
func (b *buffer) wordEnd(p Pos, big bool) Pos {
	q, ok := b.next(p)
	for ok && b.classAt(q, big) == space {
		q, ok = b.next(q)
	}
	c := b.classAt(q, big)
	for ok {
		n, more := b.next(q)
		if !more || b.classAt(n, big) != c || n.Line != q.Line {
			break
		}
		q = n
	}
	return q
}

// wordBackward is b: the start of the current or previous word.
func (b *buffer) wordBackward(p Pos, big bool) Pos {
	q, ok := b.prev(p)
	for ok && b.classAt(q, big) == space {
		if len(b.lines[q.Line]) == 0 && q.Line != p.Line {
			return q
		}
		q, ok = b.prev(q)
	}
	c := b.classAt(q, big)
	for ok {
		n, more := b.prev(q)
		if !more || b.classAt(n, big) != c || n.Line != q.Line {
			break
		}
		q = n
	}
	return q
}

// textObject returns the range of a text object (iw, a", i() around p.
func (b *buffer) textObject(p Pos, around bool, obj rune) (Pos, Pos, bool) {
	switch obj {
	case 'w', 'W':
		return b.wordObject(p, around, obj == 'W')
	case '"', '\'', '`':
		return b.quoteObject(p, around, obj)
	}
	pairs := map[rune][2]rune{
		'(': {'(', ')'}, ')': {'(', ')'}, 'b': {'(', ')'},
		'{': {'{', '}'}, '}': {'{', '}'}, 'B': {'{', '}'},
		'[': {'[', ']'}, ']': {'[', ']'},
		'<': {'<', '>'}, '>': {'<', '>'},
	}
	if pr, ok := pairs[obj]; ok {
		return b.pairObject(p, around, pr[0], pr[1])
	}
	return p, p, false
}

func (b *buffer) wordObject(p Pos, around, big bool) (Pos, Pos, bool) {
	l := b.lines[p.Line]
	if len(l) == 0 {
		// ciw on an empty line still changes it
		return p, p, !around
	}
	col := min(p.Col, len(l)-1)
	c := classOf(l[col], big)
	s, e := col, col+1
	for s > 0 && classOf(l[s-1], big) == c {
		s--
	}
	for e < len(l) && classOf(l[e], big) == c {
		e++
	}
	if around {
		switch {
		case c == space:
			// aw on blanks: the blanks and the word after them
			if e < len(l) {
				c2 := classOf(l[e], big)
				for e < len(l) && classOf(l[e], big) == c2 {
					e++
				}
			}
		case e < len(l) && classOf(l[e], big) == space:
			for e < len(l) && classOf(l[e], big) == space {
				e++
			}
		default:
			for s > 0 && classOf(l[s-1], big) == space {
				s--
			}
		}
	}
	return Pos{p.Line, s}, Pos{p.Line, e}, true
}

// nextWordObject extends a counted iw or aw from its end p. A line break
// does not count.
func (b *buffer) nextWordObject(p Pos, around, big bool) (Pos, bool) {
	if p.Col >= len(b.lines[p.Line]) {
		if p.Line == len(b.lines)-1 {
			return p, false
		}
		next := Pos{p.Line + 1, firstNonBlank(b.lines[p.Line+1])}
		if around && next.Col >= len(b.lines[next.Line]) {
			return next, true
		}
		if !around {
			next.Col = 0
		}
		p = next
	}
	_, e, ok := b.wordObject(p, around, big)
	return e, ok
}

func (b *buffer) quoteObject(p Pos, around bool, q rune) (Pos, Pos, bool) {
	l := b.lines[p.Line]
	var quotes []int
	for i, r := range l {
		if r == q && (i == 0 || l[i-1] != '\\') {
			quotes = append(quotes, i)
		}
	}
	for i := 0; i+1 < len(quotes); i += 2 {
		s, e := quotes[i], quotes[i+1]
		if p.Col <= e || i+2 >= len(quotes) {
			if p.Col > e && i+2 >= len(quotes) {
				return p, p, false
			}
			if around {
				// a" takes the blanks after, or else before
				end := e + 1
				for end < len(l) && (l[end] == ' ' || l[end] == '\t') {
					end++
				}
				if end == e+1 {
					for s > 0 && (l[s-1] == ' ' || l[s-1] == '\t') {
						s--
					}
				}
				return Pos{p.Line, s}, Pos{p.Line, end}, true
			}
			return Pos{p.Line, s + 1}, Pos{p.Line, e}, true
		}
	}
	return p, p, false
}

func (b *buffer) pairObject(p Pos, around bool, open, closing rune) (Pos, Pos, bool) {
	// Find the opening bracket enclosing p, then its match.
	depth := 0
	start, ok := p, true
	if b.at(start) == closing {
		start, ok = b.prev(start)
	}
	for ; ok; start, ok = b.prev(start) {
		switch b.at(start) {
		case closing:
			depth++
		case open:
			if depth == 0 {
				goto found
			}
			depth--
		}
	}
	return p, p, false
found:
	depth = 0
	end, more := b.next(start)
	for ; more; end, more = b.next(end) {
		switch b.at(end) {
		case open:
			depth++
		case closing:
			if depth == 0 {
				if around {
					return start, Pos{end.Line, end.Col + 1}, true
				}
				inner, _ := b.next(start)
				// {, then lines, then }: the inner object is the lines between
				if start.Col == len(b.lines[start.Line])-1 && end.Line > start.Line+1 &&
					end.Col <= firstNonBlank(b.lines[end.Line]) {
					return Pos{start.Line + 1, 0}, Pos{end.Line - 1, len(b.lines[end.Line-1])}, true
				}
				return inner, end, true
			}
			depth--
		}
	}
	return p, p, false
}
