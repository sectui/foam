// Package draw renders frames, scrollbars, empty states and the dimmed
// background under a modal.
package draw

import (
	"image/color"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/theme"
)

// Fit crops and pads s to exactly w×h cells.
func Fit(s string, w, h int) string {
	return strings.Join(FitLines(s, w, h), "\n")
}

func FitLines(s string, w, h int) []string {
	src := strings.Split(s, "\n")
	out := make([]string, max(h, 0))
	for i := range out {
		line := ""
		if i < len(src) {
			line = src[i]
		}
		out[i] = Pad(line, w)
	}
	return out
}

// Pad crops or pads a line to exactly w cells.
func Pad(line string, w int) string {
	line = ansi.Truncate(line, max(w, 0), "")
	return line + strings.Repeat(" ", max(w-ansi.StringWidth(line), 0))
}

// Scroll describes a vertical scrollbar: Total rows, Visible of them from Offset.
type Scroll struct {
	Total, Visible, Offset int
}

// Thumb returns the rows [start, end) of the thumb on a track of n rows, or
// ok=false when everything fits.
func (s Scroll) Thumb(n int) (start, end int, ok bool) {
	if s.Total <= s.Visible || n <= 0 || s.Visible <= 0 {
		return 0, 0, false
	}
	size := max(n*s.Visible/s.Total, 1)
	maxOff := s.Total - s.Visible
	start = (n - size) * min(max(s.Offset, 0), maxOff) / maxOff
	return start, start + size, true
}

type FrameOpts struct {
	Title  string // top left
	Right  string // top right
	Footer string // bottom right
	Active bool
	Scroll Scroll // drawn on the right border when Total > Visible
}

// Frame draws a themed border of exactly w×h around content.
func Frame(content string, w, h int, o FrameOpts) string {
	if w < 2 || h < 2 {
		return Fit(content, w, h)
	}
	b := theme.Border()
	bs := theme.Muted().Style()
	ts := theme.Fg().Style()
	if o.Active {
		bs = theme.Primary().Style()
		ts = theme.Primary().Style().Bold(true)
	}
	inner := w - 2

	top := edge(b.TopLeft, b.Top, b.TopRight, inner, o.Title, o.Right, bs, ts)
	bottom := edge(b.BottomLeft, b.Bottom, b.BottomRight, inner, "", o.Footer, bs, theme.Subtle().Style())

	start, end, bar := o.Scroll.Thumb(h - 2)
	thumb := theme.Fg().Style()
	lines := make([]string, 0, h)
	lines = append(lines, top)
	for i, l := range FitLines(content, inner, h-2) {
		right := bs.Render(b.Right)
		if bar && i >= start && i < end {
			right = thumb.Render("█")
		}
		lines = append(lines, bs.Render(b.Left)+l+right)
	}
	lines = append(lines, bottom)
	return strings.Join(lines, "\n")
}

func edge(l, fill, r string, inner int, left, right string, bs, ts lipgloss.Style) string {
	if left != "" && inner >= 5 {
		left = " " + ansi.Truncate(left, inner-3, "…") + " "
	} else {
		left = ""
	}
	lw := ansi.StringWidth(left)
	if right != "" {
		right = " " + ansi.Truncate(right, max(inner-lw-4, 0), "…") + " "
		if inner-lw-ansi.StringWidth(right) < 2 {
			right = ""
		}
	}
	rw := ansi.StringWidth(right)
	if lw == 0 {
		return bs.Render(l+strings.Repeat(fill, inner-rw-min(rw, 1))) + ts.Render(right) + bs.Render(strings.Repeat(fill, min(rw, 1))+r)
	}
	mid := max(inner-1-lw-rw-min(rw, 1), 0)
	return bs.Render(l+fill) + ts.Render(left) + bs.Render(strings.Repeat(fill, mid)) +
		ts.Render(right) + bs.Render(strings.Repeat(fill, min(rw, 1))+r)
}

// Hatch is what an empty pane shows. U+2571 joins across cells where "/"
// would not.
func Hatch(w, h int, msg string) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	hs := theme.Selection().Style()
	row := hs.Render(strings.Repeat("╱", w))
	lines := make([]string, h)
	for i := range lines {
		lines[i] = row
	}
	if msg != "" {
		msg = " " + ansi.Truncate(msg, max(w-4, 0), "…") + " "
		mw := ansi.StringWidth(msg)
		pad := max((w-mw)/2, 0)
		lines[h/2] = hs.Render(strings.Repeat("╱", pad)) + theme.Subtle().Render(msg) + hs.Render(strings.Repeat("╱", max(w-pad-mw, 0)))
	}
	return strings.Join(lines, "\n")
}

func Rule(w int) string { return theme.Muted().Render(strings.Repeat("─", max(w, 0))) }

// Selected renders a full-width selected row, its own styles stripped.
func Selected(line string, w int, focused bool) string {
	bg := theme.Selection()
	if !focused {
		bg = theme.Surface()
	}
	return lipgloss.NewStyle().Background(bg.Value()).Foreground(theme.Fg().Value()).Render(Pad(ansi.Strip(line), w))
}

// Dim fades c halfway to the background, under a modal.
func Dim(c *lipgloss.Canvas) {
	bg := theme.Bg().Value()
	faded := map[[4]uint32]color.Color{}
	fade := func(col color.Color) color.Color {
		r, g, b, a := col.RGBA()
		k := [4]uint32{r, g, b, a}
		if f, ok := faded[k]; ok {
			return f
		}
		f := lipgloss.Blend1D(3, col, bg)[1]
		faded[k] = f
		return f
	}
	for y := range c.Height() {
		for x := range c.Width() {
			cell := c.CellAt(x, y)
			if cell == nil || cell.Width == 0 {
				// the second half of a wide character: writing it would
				// erase the character
				continue
			}
			d := *cell
			if d.Style.Fg == nil {
				d.Style.Fg = theme.Fg().Value()
			}
			d.Style.Fg = fade(d.Style.Fg)
			if d.Style.Bg != nil {
				d.Style.Bg = fade(d.Style.Bg)
			}
			c.SetCell(x, y, &d)
		}
	}
}

func ProgressBar(done, total, w int) string {
	if w <= 0 {
		return ""
	}
	fill := 0
	if total > 0 {
		fill = min(w*done/total, w)
	}
	return theme.Primary().Render(strings.Repeat("━", fill)) + theme.Muted().Render(strings.Repeat("━", w-fill))
}

// FillRow fills the cells of row y that have no background, so that a bar
// stays even whatever its parts are styled with.
func FillRow(c *lipgloss.Canvas, y int, bg theme.Color) {
	if y < 0 || y >= c.Height() {
		return
	}
	col := bg.Value()
	for x := range c.Width() {
		cell := c.CellAt(x, y)
		if cell == nil || cell.Width == 0 {
			continue
		}
		d := *cell
		if d.Style.Bg == nil {
			d.Style.Bg = col
			c.SetCell(x, y, &d)
		}
	}
}

// Clean makes outside text safe to draw: control characters and invalid
// UTF-8 become "·", tabs a space.
func Clean(s string) string {
	ok := utf8.ValidString(s)
	if ok {
		for _, r := range s {
			if isControl(r) || r == '\t' {
				ok = false
				break
			}
		}
	}
	if ok {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\t':
			b.WriteByte(' ')
		case r == utf8.RuneError || isControl(r):
			b.WriteRune('·')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isControl(r rune) bool { return r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f }
