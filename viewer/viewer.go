// Package viewer shows read-only text: scrolling both ways, wrapping, and a
// search that highlights its matches.
package viewer

import (
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anotherhadi/foam/commands"
	"github.com/anotherhadi/foam/draw"
	"github.com/anotherhadi/foam/theme"
)

type match struct{ line, start, end int } // cells

type Model struct {
	Wrap bool

	plain  []string
	styled []string
	widths []int

	w, h int
	x, y int // y counts screen rows, wrapped ones included

	query   string
	matches []match
	cur     int

	// What View draws, kept until the content, the size, the search or the
	// scroll changes: a line of megabytes is not cut again on every frame.
	rows      map[int][]string // wrapped rows of a line
	cuts      map[int]string   // the visible part of a line, unwrapped
	cacheW    int
	cacheX    int
	cacheWrap bool
}

func (m *Model) invalidate() { m.rows, m.cuts = nil, nil }

func (m *Model) checkCache() {
	if m.cacheW != m.w || m.cacheWrap != m.Wrap {
		m.invalidate()
		m.cacheW, m.cacheWrap = m.w, m.Wrap
	}
	if m.cacheX != m.x {
		m.cuts = nil
		m.cacheX = m.x
	}
	if m.rows == nil {
		m.rows = map[int][]string{}
	}
	if m.cuts == nil {
		m.cuts = map[int]string{}
	}
}

func (m *Model) wrapped(i int) []string {
	m.checkCache()
	if r, ok := m.rows[i]; ok {
		return r
	}
	r := splitRows(m.line(i), m.w)
	for k, row := range r {
		r[k] = fit(row, m.w)
	}
	m.rows[i] = r
	return r
}

// SetContent replaces the text. styled, when not nil, holds the same lines
// with colors; it must have as many lines as plain, and the same widths.
func (m *Model) SetContent(plain, styled []string) {
	m.plain = make([]string, len(plain))
	for i, l := range plain {
		m.plain[i] = Sanitize(l)
	}
	m.styled = styled
	if len(styled) != len(plain) {
		m.styled = nil
	}
	m.widths = make([]int, len(m.plain))
	for i, l := range m.plain {
		m.widths[i] = ansi.StringWidth(l)
	}
	m.invalidate()
	m.search()
	m.clamp()
}

// Sanitize expands tabs and makes control characters visible, so that the
// width of a line is what the terminal shows.
func Sanitize(s string) string {
	if utf8.ValidString(s) && !strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f }) {
		return s
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		switch {
		case r == '\t':
			n := 4 - col%4
			b.WriteString(strings.Repeat(" ", n))
			col += n
			continue
		case r == '\r':
			continue
		case r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f || r == utf8.RuneError:
			r = '·'
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

func (m *Model) Lines() []string { return m.plain }

func (m *Model) SetSize(w, h int) {
	m.w, m.h = max(w, 0), max(h, 0)
	m.clamp()
}

func (m *Model) Top() { m.x, m.y = 0, 0 }

func (m *Model) rowsOf(i int) int {
	if !m.Wrap || m.w <= 0 {
		return 1
	}
	if m.widths[i] <= m.w {
		return 1
	}
	return len(m.wrapped(i)) // wide characters may not split evenly
}

func (m *Model) totalRows() int {
	if !m.Wrap {
		return len(m.plain)
	}
	n := 0
	for i := range m.plain {
		n += m.rowsOf(i)
	}
	return n
}

func (m *Model) maxWidth() int {
	w := 0
	for _, x := range m.widths {
		w = max(w, x)
	}
	return w
}

func (m *Model) clamp() {
	m.y = max(min(m.y, m.totalRows()-m.h), 0)
	if m.Wrap {
		m.x = 0
	}
	m.x = max(min(m.x, m.maxWidth()-m.w), 0)
}

func (m *Model) scroll(dy, dx int) tea.Cmd {
	m.y += dy
	m.x += dx
	m.clamp()
	return nil
}

func (m *Model) Scroll() draw.Scroll {
	return draw.Scroll{Total: m.totalRows(), Visible: m.h, Offset: m.y}
}

// Commands are bound under viewer.*.
func (m *Model) Commands() []commands.Command {
	half := func() int { return max(m.h/2, 1) }
	page := func() int { return max(m.h-1, 1) }
	return []commands.Command{
		{ID: "viewer.down", Help: "scroll", Run: func() tea.Cmd { return m.scroll(1, 0) }},
		{ID: "viewer.up", Run: func() tea.Cmd { return m.scroll(-1, 0) }},
		{ID: "viewer.left", When: func() bool { return !m.Wrap }, Run: func() tea.Cmd { return m.scroll(0, -4) }},
		{ID: "viewer.right", When: func() bool { return !m.Wrap }, Run: func() tea.Cmd { return m.scroll(0, 4) }},
		{ID: "viewer.page-down", Run: func() tea.Cmd { return m.scroll(page(), 0) }},
		{ID: "viewer.page-up", Run: func() tea.Cmd { return m.scroll(-page(), 0) }},
		{ID: "viewer.half-down", Run: func() tea.Cmd { return m.scroll(half(), 0) }},
		{ID: "viewer.half-up", Run: func() tea.Cmd { return m.scroll(-half(), 0) }},
		{ID: "viewer.top", Run: func() tea.Cmd { m.y = 0; return nil }},
		{ID: "viewer.bottom", Run: func() tea.Cmd { m.y = 1 << 30; m.clamp(); return nil }},
	}
}

// Mouse scrolls on the wheel (shift+wheel or a horizontal wheel sideways).
func (m *Model) Mouse(msg tea.MouseMsg) tea.Cmd {
	w, ok := msg.(tea.MouseWheelMsg)
	if !ok {
		return nil
	}
	switch {
	case w.Button == tea.MouseWheelLeft || (w.Button == tea.MouseWheelUp && w.Mod&tea.ModShift != 0):
		return m.scroll(0, -4)
	case w.Button == tea.MouseWheelRight || (w.Button == tea.MouseWheelDown && w.Mod&tea.ModShift != 0):
		return m.scroll(0, 4)
	case w.Button == tea.MouseWheelUp:
		return m.scroll(-3, 0)
	case w.Button == tea.MouseWheelDown:
		return m.scroll(3, 0)
	}
	return nil
}

// Search highlights every match of q (case-insensitive) and jumps to the
// first one at or after the top of the screen. It returns the match count.
func (m *Model) Search(q string) int {
	m.query = q
	m.search()
	if len(m.matches) > 0 {
		top := m.lineAt(m.y)
		m.cur = 0
		for i, mt := range m.matches {
			if mt.line >= top {
				m.cur = i
				break
			}
		}
		m.reveal()
	}
	return len(m.matches)
}

func (m *Model) search() {
	m.invalidate()
	m.matches, m.cur = nil, 0
	if m.query == "" {
		return
	}
	q := strings.ToLower(m.query)
	for i, l := range m.plain {
		low := strings.ToLower(l)
		if len(low) != len(l) {
			low = l
		}
		for off := 0; ; {
			j := strings.Index(low[off:], q)
			if j < 0 {
				break
			}
			start := ansi.StringWidth(l[:off+j])
			m.matches = append(m.matches, match{i, start, start + ansi.StringWidth(l[off+j:off+j+len(q)])})
			off += j + max(len(q), 1)
		}
	}
}

// Next moves to the next match (d=1) or the previous one (d=-1).
func (m *Model) Next(d int) {
	if n := len(m.matches); n > 0 {
		m.invalidate() // the current match changes color
		m.cur = ((m.cur+d)%n + n) % n
		m.reveal()
	}
}

// Status is the current match (from 1) and the match count; 0, 0 without any.
func (m *Model) Status() (cur, total int) {
	if len(m.matches) == 0 {
		return 0, 0
	}
	return m.cur + 1, len(m.matches)
}

func (m *Model) ClearSearch() {
	if m.query != "" || m.matches != nil {
		m.invalidate()
	}
	m.query, m.matches = "", nil
}

func (m *Model) lineAt(row int) int {
	if !m.Wrap {
		return row
	}
	for i := range m.plain {
		if row -= m.rowsOf(i); row < 0 {
			return i
		}
	}
	return len(m.plain)
}

func (m *Model) rowOf(line, col int) int {
	if !m.Wrap {
		return line
	}
	row := 0
	for i := 0; i < line && i < len(m.plain); i++ {
		row += m.rowsOf(i)
	}
	if m.w > 0 {
		row += col / m.w
	}
	return row
}

func (m *Model) reveal() {
	mt := m.matches[m.cur]
	row := m.rowOf(mt.line, mt.start)
	if row < m.y || row >= m.y+m.h {
		m.y = row - m.h/3
	}
	if !m.Wrap && (mt.start < m.x || mt.end > m.x+m.w) {
		m.x = mt.start - m.w/3
	}
	m.clamp()
}

func (m *Model) line(i int) string {
	s := m.plain[i]
	if m.styled != nil {
		s = m.styled[i]
	} else {
		s = theme.Fg().Render(s)
	}
	var rs []lipgloss.Range
	hl := lipgloss.NewStyle().Background(theme.Warning().Value()).Foreground(theme.Bg().Value())
	cur := lipgloss.NewStyle().Background(theme.Orange().Value()).Foreground(theme.Bg().Value()).Bold(true)
	for k, mt := range m.matches {
		if mt.line != i {
			continue
		}
		st := hl
		if k == m.cur {
			st = cur
		}
		rs = append(rs, lipgloss.NewRange(mt.start, mt.end, st))
	}
	return lipgloss.StyleRanges(s, rs...)
}

func (m *Model) View() string {
	if m.w <= 0 || m.h <= 0 {
		return ""
	}
	m.checkCache()
	out := make([]string, 0, m.h)
	if !m.Wrap {
		for i := m.y; i < len(m.plain) && len(out) < m.h; i++ {
			c, ok := m.cuts[i]
			if !ok {
				c = ansi.Cut(m.line(i), m.x, m.x+m.w)
				// a wide character cut by the left edge is kept whole: drop it
				if over := ansi.StringWidth(c) - m.w; over > 0 {
					c = fit(ansi.TruncateLeft(c, over+1, ""), m.w)
				}
				m.cuts[i] = c
			}
			out = append(out, c)
		}
		return strings.Join(out, "\n")
	}
	skip := m.y
	for i := range m.plain {
		n := m.rowsOf(i)
		if skip >= n {
			skip -= n
			continue
		}
		rows := m.wrapped(i)
		for r := skip; r < len(rows) && len(out) < m.h; r++ {
			out = append(out, rows[r])
		}
		skip = 0
		if len(out) >= m.h {
			break
		}
	}
	return strings.Join(out, "\n")
}

// fit keeps a row within w cells, even when a single character is wider.
func fit(row string, w int) string {
	if ansi.StringWidth(row) <= w {
		return row
	}
	if t := ansi.Truncate(row, w, ""); ansi.StringWidth(t) <= w {
		return t
	}
	return ""
}
