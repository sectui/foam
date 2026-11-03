package shot

import (
	"bufio"
	"fmt"
	"image/color"
	"io"
	"math"
	"path/filepath"
	"strings"
)

func renderSVG(w io.Writer, g Grid, o Options) error {
	px := o.size() * 96 / 72
	cw, ch, base := int(math.Round(px*0.6)), int(math.Round(px*1.25)), int(math.Round(px*0.95))
	name := o.Font
	if fs, err := loadFonts(o.Font, px); err == nil {
		cw, ch, base = fs.cell()
		if fs.name != "" {
			name = fs.name
		}
	} else if isFile(name) {
		name = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	}
	family := "monospace"
	if name != "" && name != "monospace" {
		family = "'" + strings.NewReplacer("'", "", "<", "", "&", "").Replace(name) + "', monospace"
	}
	light := math.Max(1, math.Round(px/14))
	W, H := g.Width()*cw, g.Height()*ch

	b := bufio.NewWriter(w)
	fmt.Fprintf(b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n", W, H, W, H)
	fmt.Fprintf(b, "<style>text{font-family:%s;font-size:%.2fpx;white-space:pre}.b{font-weight:bold}.i{font-style:italic}</style>\n", family, px)

	type run struct {
		x0, x1 int
		c      cell
	}
	rows := make([][]run, g.Height())
	cells(g, o, func(x, y int, c cell) {
		rows[y] = append(rows[y], run{x, x + c.width, c})
	})

	for y, row := range rows {
		for i := 0; i < len(row); {
			j := i + 1
			for j < len(row) && row[j].c.bg == row[i].c.bg {
				j++
			}
			fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`+"\n",
				row[i].x0*cw, y*ch, (row[j-1].x1-row[i].x0)*cw, ch, hex(row[i].c.bg))
			i = j
		}
	}

	for y, row := range rows {
		var text []run
		flush := func() {
			if len(text) == 0 {
				return
			}
			c := text[0].c
			var xs []string
			var s strings.Builder
			for _, t := range text {
				xs = append(xs, fmt.Sprint(t.x0*cw))
				s.WriteString(escape(t.c.text))
			}
			class := ""
			if c.bold {
				class += " b"
			}
			if c.italic {
				class += " i"
			}
			if class != "" {
				class = ` class="` + class[1:] + `"`
			}
			fmt.Fprintf(b, `<text x="%s" y="%d" fill="%s"%s>%s</text>`+"\n", strings.Join(xs, " "), y*ch+base, hex(c.fg), class, s.String())
			text = text[:0]
		}
		for _, r := range row {
			c := r.c
			x0, x1, y0 := r.x0*cw, r.x1*cw, y*ch
			line := func(at int) {
				fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%g" fill="%s"/>`+"\n", x0, at, x1-x0, light, hex(c.fg))
			}
			if c.underline {
				line(y0 + min(base+int(light), ch-int(light)))
			}
			if c.strike {
				line(y0 + ch/2)
			}
			runes := []rune(c.text)
			if c.hidden || c.text == "" || c.text == " " {
				flush()
				continue
			}
			if len(runes) == 1 {
				if s, ok := glyph(runes[0], float64(x1-x0), float64(ch), light); ok {
					flush()
					var d strings.Builder
					for _, p := range s.polys {
						for k, q := range p {
							op := "L"
							if k == 0 {
								op = "M"
							}
							fmt.Fprintf(&d, "%s%g %g", op, round2(float64(x0)+q.x), round2(float64(y0)+q.y))
						}
						d.WriteString("Z")
					}
					fmt.Fprintf(b, `<svg x="%d" y="%d" width="%d" height="%d" viewBox="%d %d %d %d" overflow="hidden"><path d="%s" fill="%s"/></svg>`+"\n",
						x0, y0, x1-x0, ch, x0, y0, x1-x0, ch, d.String(), hex(mix(c.bg, c.fg, s.alpha)))
					continue
				}
			}
			if len(runes) != 1 || c.width != 1 {
				flush()
				text = append(text, r)
				flush()
				continue
			}
			if len(text) > 0 {
				p := text[len(text)-1]
				if p.x1 != r.x0 || p.c.fg != c.fg || p.c.bold != c.bold || p.c.italic != c.italic {
					flush()
				}
			}
			text = append(text, r)
		}
		flush()
	}
	b.WriteString("</svg>\n")
	return b.Flush()
}

func hex(c color.RGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func round2(f float64) float64 { return math.Round(f*100) / 100 }

// escape makes text safe in XML, dropping what XML cannot hold.
func escape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '&':
			b.WriteString("&amp;")
		case r < 0x20 || r == 0xfffe || r == 0xffff:
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
