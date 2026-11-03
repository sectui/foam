// Package shot renders a screen of cells to an image, as a terminal would draw
// it: PNG with real fonts, or SVG. Box drawing, blocks and powerline shapes
// are drawn as shapes, so that they join across cells.
package shot

import (
	"image/color"
	"io"

	uv "github.com/charmbracelet/ultraviolet"
)

// Grid is a screen of cells; a lipgloss.Canvas is one.
type Grid interface {
	Width() int
	Height() int
	CellAt(x, y int) *uv.Cell
}

type Options struct {
	// Font is a fontconfig family ("JetBrainsMono Nerd Font") or a font file;
	// "" is fontconfig's monospace.
	Font string
	// Size is in points at 96 DPI; 0 is 12.
	Size float64
	// Scale multiplies the pixels of a PNG; 0 is 1.
	Scale float64
	// Fg and Bg stand for the terminal's default colors.
	Fg, Bg color.Color
}

func (o Options) size() float64 {
	if o.Size <= 0 {
		return 12
	}
	return o.Size
}

func (o Options) scale() float64 {
	if o.Scale <= 0 {
		return 1
	}
	return o.Scale
}

// PNG draws g with the fonts o names.
func PNG(w io.Writer, g Grid, o Options) error {
	fs, err := loadFonts(o.Font, o.size()*o.scale()*96/72)
	if err != nil {
		return err
	}
	return renderPNG(w, g, o, fs)
}

// SVG draws g as text in o's font family; the viewer must have the font.
func SVG(w io.Writer, g Grid, o Options) error {
	return renderSVG(w, g, o)
}

// cell is a cell resolved for drawing: its colors applied, its attributes
// read.
type cell struct {
	text         string
	width        int
	fg, bg       color.RGBA
	bold, italic bool
	underline    bool
	strike       bool
	hidden       bool
}

func resolve(c *uv.Cell, o Options) cell {
	fg, bg := rgba(o.Fg, color.RGBA{0xcd, 0xd6, 0xf4, 0xff}), rgba(o.Bg, color.RGBA{0x1e, 0x1e, 0x2e, 0xff})
	r := cell{text: c.Content, width: c.Width}
	if c.Style.Fg != nil {
		fg = rgba(c.Style.Fg, fg)
	}
	if c.Style.Bg != nil {
		bg = rgba(c.Style.Bg, bg)
	}
	a := c.Style.Attrs
	if a&uv.AttrReverse != 0 {
		fg, bg = bg, fg
	}
	if a&uv.AttrFaint != 0 {
		fg = mix(fg, bg, 0.5)
	}
	r.fg, r.bg = fg, bg
	r.bold = a&uv.AttrBold != 0
	r.italic = a&uv.AttrItalic != 0
	r.strike = a&uv.AttrStrikethrough != 0
	r.hidden = a&uv.AttrConceal != 0
	r.underline = c.Style.Underline != uv.UnderlineNone
	return r
}

func rgba(c color.Color, def color.RGBA) color.RGBA {
	if c == nil {
		return def
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return def
	}
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0xff}
}

// mix blends a toward b: 0 is a, 1 is b.
func mix(a, b color.RGBA, t float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t + 0.5) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 0xff}
}

// cells walks g's cells, skipping the second half of wide ones.
func cells(g Grid, o Options, f func(x, y int, c cell)) {
	for y := range g.Height() {
		for x := 0; x < g.Width(); x++ {
			c := g.CellAt(x, y)
			if c == nil {
				f(x, y, resolve(&uv.EmptyCell, o))
				continue
			}
			if c.Width == 0 {
				continue
			}
			f(x, y, resolve(c, o))
		}
	}
}
