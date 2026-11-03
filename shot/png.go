package shot

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"

	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

func renderPNG(w io.Writer, g Grid, o Options, fs *fonts) error {
	cw, ch, base := fs.cell()
	light := math.Max(1, math.Round(fs.px/14))
	img := image.NewRGBA(image.Rect(0, 0, g.Width()*cw, g.Height()*ch))
	cellRect := func(x, y, width int) image.Rectangle {
		return image.Rect(x*cw, y*ch, (x+width)*cw, (y+1)*ch)
	}
	cells(g, o, func(x, y int, c cell) {
		draw.Draw(img, cellRect(x, y, c.width), image.NewUniform(c.bg), image.Point{}, draw.Src)
	})
	// glyphs after every background: wide icons overflow onto the next cell
	var z vector.Rasterizer
	cells(g, o, func(x, y int, c cell) {
		r := cellRect(x, y, c.width)
		line := func(y0 int) {
			draw.Draw(img, image.Rect(r.Min.X, y0, r.Max.X, y0+int(light)), image.NewUniform(c.fg), image.Point{}, draw.Src)
		}
		if c.underline {
			line(r.Min.Y + min(base+int(light), ch-int(light)))
		}
		if c.strike {
			line(r.Min.Y + ch/2)
		}
		if c.hidden || c.text == "" || c.text == " " {
			return
		}
		runes := []rune(c.text)
		if len(runes) == 1 {
			if s, ok := glyph(runes[0], float64(r.Dx()), float64(ch), light); ok {
				z.Reset(r.Dx(), ch)
				for _, p := range s.polys {
					z.MoveTo(float32(p[0].x), float32(p[0].y))
					for _, q := range p[1:] {
						z.LineTo(float32(q.x), float32(q.y))
					}
					z.ClosePath()
				}
				z.Draw(img, r, image.NewUniform(mix(c.bg, c.fg, s.alpha)), image.Point{})
				return
			}
		}
		st := regular
		switch {
		case c.bold && c.italic:
			st = boldItalic
		case c.bold:
			st = bold
		case c.italic:
			st = italic
		}
		dot := fixed.P(r.Min.X, r.Min.Y+base)
		src := image.NewUniform(c.fg)
		for i, ru := range runes {
			if ru == 0x200d || ru >= 0xfe00 && ru <= 0xfe0f {
				continue
			}
			f, fake := fs.pick(ru, st)
			if f == nil {
				if i == 0 {
					tofu(img, r.Inset(int(light)*2), int(light), mix(c.bg, c.fg, 0.5))
				}
				continue
			}
			dr, mask, mp, adv, ok := f.face.Glyph(dot, ru)
			if !ok {
				continue
			}
			draw.DrawMask(img, dr, src, image.Point{}, mask, mp, draw.Over)
			if fake {
				draw.DrawMask(img, dr.Add(image.Pt(int(light), 0)), src, image.Point{}, mask, mp, draw.Over)
			}
			dot.X += adv
		}
	})
	return png.Encode(w, img)
}

// tofu outlines r: what no font could draw.
func tofu(img *image.RGBA, r image.Rectangle, t int, c color.RGBA) {
	u := image.NewUniform(c)
	for _, s := range []image.Rectangle{
		{r.Min, image.Pt(r.Max.X, r.Min.Y+t)}, {image.Pt(r.Min.X, r.Max.Y-t), r.Max},
		{r.Min, image.Pt(r.Min.X+t, r.Max.Y)}, {image.Pt(r.Max.X-t, r.Min.Y), r.Max},
	} {
		draw.Draw(img, s, u, image.Point{}, draw.Src)
	}
}
