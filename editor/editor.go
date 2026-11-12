// Package editor draws a vim.Editor: wrapped lines colored by the app, the
// selection, the cursor, and a scroll that follows it.
package editor

import (
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anotherhadi/foam/draw"
	"github.com/anotherhadi/foam/theme"
	"github.com/anotherhadi/foam/vim"
)

type Model struct {
	*vim.Editor

	// Style colors the lines; it must keep their width. nil draws them plain.
	Style func(lines []string) []string
	// HideCursor draws no cursor, the selection still (for screenshots).
	HideCursor bool

	w, h int
	top  int // first screen row shown

	cacheVersion uint64
	cacheW       int
	cacheStyled  []string
	cacheWidths  []int
	// cacheBreaks holds the first cell of each screen row of a line: a wide
	// character that does not fit at the end of a row starts the next one.
	cacheBreaks [][]int
}

func New(text string) *Model { return &Model{Editor: vim.New(text)} }

func (m *Model) SetSize(w, h int) {
	m.w, m.h = max(w, 1), max(h, 0)
	m.follow()
}

func shown(s string) string {
	if !strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f || r == utf8.RuneError }) {
		return s
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return '›'
		case r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f || r == utf8.RuneError:
			return '·'
		}
		return r
	}, s)
}

func (m *Model) refresh() {
	if m.cacheStyled != nil && m.Version() == m.cacheVersion && m.cacheW == m.w {
		return
	}
	lines := m.Lines()
	for i, l := range lines {
		lines[i] = shown(l)
	}
	m.cacheWidths = make([]int, len(lines))
	m.cacheBreaks = make([][]int, len(lines))
	for i, l := range lines {
		m.cacheWidths[i] = ansi.StringWidth(l)
		m.cacheBreaks[i] = breaks(l, m.w)
	}
	m.cacheW = m.w
	if m.Style != nil {
		m.cacheStyled = m.Style(lines)
	} else {
		m.cacheStyled = make([]string, len(lines))
		for i, l := range lines {
			m.cacheStyled[i] = theme.Fg().Render(l)
		}
	}
	m.cacheVersion = m.Version()
}

// Restyle drops the colored lines, for a Style that changed.
func (m *Model) Restyle() { m.cacheStyled = nil }

// rowsOf counts the screen rows of line i; a cell is kept after the last
// character for the cursor.
func (m *Model) rowsOf(i int) int { return len(m.cacheBreaks[i]) }

func breaks(line string, w int) []int {
	out := []int{0}
	pos := 0
	for _, r := range line {
		rw := ansi.StringWidth(string(r))
		if start := out[len(out)-1]; pos > start && pos-start+rw > w {
			out = append(out, pos)
		}
		pos += rw
	}
	// the cell kept after the last character, for the cursor
	if pos-out[len(out)-1]+1 > w {
		out = append(out, pos)
	}
	return out
}

// rowAt is the screen row of cell x in line i, and where in that row x is.
func (m *Model) rowAt(i, x int) (row, col int) {
	b := m.cacheBreaks[i]
	for row+1 < len(b) && b[row+1] <= x {
		row++
	}
	return row, x - b[row]
}

func (m *Model) cells(i, col int) int {
	l := []rune(m.Line(i))
	return ansi.StringWidth(shown(string(l[:min(col, len(l))])))
}

func (m *Model) rowOf(p vim.Pos) int {
	m.refresh()
	row := 0
	for i := 0; i < p.Line && i < len(m.cacheWidths); i++ {
		row += m.rowsOf(i)
	}
	r, _ := m.rowAt(p.Line, m.cells(p.Line, p.Col))
	return row + r
}

func (m *Model) totalRows() int {
	m.refresh()
	n := 0
	for i := range m.cacheWidths {
		n += m.rowsOf(i)
	}
	return n
}

func (m *Model) follow() {
	if m.h <= 0 {
		return
	}
	r := m.rowOf(m.Cursor())
	if r < m.top {
		m.top = r
	}
	if r >= m.top+m.h {
		m.top = r - m.h + 1
	}
	m.top = max(min(m.top, m.totalRows()-m.h), 0)
}

func (m *Model) Scroll() draw.Scroll {
	return draw.Scroll{Total: m.totalRows(), Visible: m.h, Offset: m.top}
}

func (m *Model) Key(msg tea.KeyPressMsg) bool {
	// The scrolling keys are the view's, not the engine's: it knows nothing
	// of how tall the text is drawn. In insert mode they are vim's own.
	if m.Mode() != vim.Insert && m.Pending() == "" {
		if d, ok := m.scrollDelta(msg.String()); ok {
			m.scrollRows(d)
			return true
		}
	}
	ok := m.Editor.Key(msg.String(), msg.Key().Text)
	m.follow()
	return ok
}

// scrollDelta is how many screen rows a key scrolls by, as vim does: half a
// screen for ctrl+d and ctrl+u, one short of a screen for the rest.
func (m *Model) scrollDelta(name string) (int, bool) {
	half, full := max(m.h/2, 1), max(m.h-2, 1)
	switch name {
	case "ctrl+d":
		return half, true
	case "ctrl+u":
		return -half, true
	case "ctrl+f", "pgdown":
		return full, true
	case "ctrl+b", "pgup":
		return -full, true
	}
	return 0, false
}

// scrollRows moves the cursor d screen rows and takes the view along, keeping
// the cursor where it is on screen while there is text to scroll.
func (m *Model) scrollRows(d int) {
	m.refresh()
	total := m.totalRows()
	if m.h <= 0 || total == 0 {
		return
	}
	cur := m.Cursor()
	_, cell := m.rowAt(cur.Line, m.cells(cur.Line, cur.Col))
	row := min(max(m.rowOf(cur)+d, 0), total-1)
	m.top = max(min(m.top+d, total-m.h), 0)
	m.SetCursor(m.posAt(cell, row-m.top))
	m.follow()
}

func (m *Model) Paste(s string) {
	m.Editor.Paste(s)
	m.follow()
}

func (m *Model) SetText(s string) {
	m.Editor.SetText(s)
	m.follow()
}

func (m *Model) Replace(a, b vim.Pos, s string) vim.Pos {
	end := m.Editor.Replace(a, b, s)
	m.follow()
	return end
}

func (m *Model) ReplaceInsert(a, b vim.Pos, before, after string) {
	m.Editor.ReplaceInsert(a, b, before, after)
	m.follow()
}

func (m *Model) GotoLine(i int) {
	m.SetCursor(vim.Pos{Line: i})
	m.follow()
}

func (m *Model) posAt(x, y int) vim.Pos {
	m.refresh()
	row := m.top + max(y, 0)
	for i := range m.cacheWidths {
		n := m.rowsOf(i)
		if row < n {
			b := m.cacheBreaks[i]
			target := b[row] + max(x, 0)
			if row+1 < len(b) {
				target = min(target, b[row+1]-1)
			}
			col, n := 0, len([]rune(m.Line(i)))
			for col < n && m.cells(i, col+1) <= target {
				col++
			}
			return vim.Pos{Line: i, Col: col}
		}
		row -= n
	}
	last := m.LineCount() - 1
	return vim.Pos{Line: last, Col: len([]rune(m.Line(last)))}
}

// Mouse scrolls on the wheel, taking the cursor along, and moves the cursor
// to a click. Coordinates are relative to the text area.
func (m *Model) Mouse(msg tea.MouseMsg) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		d := 3
		if msg.Button == tea.MouseWheelUp {
			d = -3
		} else if msg.Button != tea.MouseWheelDown {
			return
		}
		m.top = max(min(m.top+d, m.totalRows()-m.h), 0)
		r := m.rowOf(m.Cursor())
		switch {
		case r < m.top:
			m.SetCursor(m.posAt(0, 0))
		case r >= m.top+m.h:
			m.SetCursor(m.posAt(0, m.h-1))
		}
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			m.SetCursor(m.posAt(msg.X, msg.Y))
		}
	}
}

// CursorPosition is where the terminal cursor goes in insert mode, relative
// to the text area; ok is false in the other modes, which draw a block.
func (m *Model) CursorPosition() (x, y int, ok bool) {
	if m.Mode() != vim.Insert {
		return 0, 0, false
	}
	c := m.Cursor()
	m.refresh()
	_, x = m.rowAt(c.Line, m.cells(c.Line, c.Col))
	y = m.rowOf(c) - m.top
	return x, y, y >= 0 && y < m.h
}

func (m *Model) View(focused bool) string {
	if m.h <= 0 {
		return ""
	}
	m.refresh()
	m.follow()
	sel := lipgloss.NewStyle().Background(theme.Selection().Value())
	block := lipgloss.NewStyle().Reverse(true)
	if !focused {
		block = lipgloss.NewStyle().Background(theme.Muted().Value())
	}
	start, end, hasSel := m.Selection()
	c := m.Cursor()

	out := make([]string, 0, m.h)
	skip := m.top
	for i := range m.cacheWidths {
		n := m.rowsOf(i)
		if skip >= n {
			skip -= n
			continue
		}
		line := m.cacheStyled[i] + " " // the cell kept for the cursor
		var ranges []lipgloss.Range
		if hasSel && i >= start.Line && i <= end.Line {
			a, b := 0, m.cacheWidths[i]+1
			if i == start.Line {
				a = m.cells(i, start.Col)
			}
			if i == end.Line {
				b = m.cells(i, end.Col)
			}
			if c.Line == i && !m.HideCursor {
				ranges = appendSplit(ranges, a, b, m.cells(i, c.Col), sel)
			} else if b > a {
				ranges = append(ranges, lipgloss.NewRange(a, b, sel))
			}
		}
		if c.Line == i && m.Mode() != vim.Insert && !m.HideCursor {
			x := m.cells(i, c.Col)
			ranges = append(ranges, lipgloss.NewRange(x, x+1, block))
		}
		sortRanges(ranges)
		line = lipgloss.StyleRanges(line, ranges...)
		b := m.cacheBreaks[i]
		for r := skip; r < n && len(out) < m.h; r++ {
			end := m.cacheWidths[i] + 1
			if r+1 < n {
				end = b[r+1]
			}
			row := ansi.Cut(line, b[r], end)
			if ansi.StringWidth(row) > m.w {
				// a character wider than the whole pane
				row = ansi.Truncate(row, m.w, "")
			}
			out = append(out, row)
		}
		skip = 0
		if len(out) >= m.h {
			break
		}
	}
	return strings.Join(out, "\n")
}

func appendSplit(rs []lipgloss.Range, a, b, x int, st lipgloss.Style) []lipgloss.Range {
	if x > a && x < b {
		return append(rs, lipgloss.NewRange(a, x, st), lipgloss.NewRange(x+1, b, st))
	}
	if x == a {
		a++
	}
	if x == b-1 {
		b--
	}
	if b > a {
		rs = append(rs, lipgloss.NewRange(a, b, st))
	}
	return rs
}

func sortRanges(rs []lipgloss.Range) {
	for i := 1; i < len(rs); i++ {
		for j := i; j > 0 && rs[j].Start < rs[j-1].Start; j-- {
			rs[j], rs[j-1] = rs[j-1], rs[j]
		}
	}
}
