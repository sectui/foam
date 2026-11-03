package shot

import "math"

type point struct{ x, y float64 }

// shape is what a box drawing, block or powerline character looks like in a
// cell of w×h pixels: polygons filled with the foreground mixed over the
// background by alpha.
type shape struct {
	polys [][]point
	alpha float64
}

func (s *shape) rect(x0, y0, x1, y1 float64) {
	if x1 > x0 && y1 > y0 {
		s.polys = append(s.polys, []point{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}})
	}
}

// stroke adds a line of thickness t along path.
func (s *shape) stroke(path []point, t float64) {
	n := len(path)
	if n < 2 {
		return
	}
	left := make([]point, n)
	right := make([]point, n)
	for i := range path {
		var dx, dy float64
		if i > 0 {
			dx, dy = unit(path[i].x-path[i-1].x, path[i].y-path[i-1].y)
		}
		if i < n-1 {
			ex, ey := unit(path[i+1].x-path[i].x, path[i+1].y-path[i].y)
			dx, dy = unit(dx+ex, dy+ey)
		}
		nx, ny := -dy*t/2, dx*t/2
		left[i] = point{path[i].x + nx, path[i].y + ny}
		right[i] = point{path[i].x - nx, path[i].y - ny}
	}
	poly := left
	for i := n - 1; i >= 0; i-- {
		poly = append(poly, right[i])
	}
	s.polys = append(s.polys, poly)
}

func unit(x, y float64) (float64, float64) {
	l := math.Hypot(x, y)
	if l == 0 {
		return 0, 0
	}
	return x / l, y / l
}

// clockwise orients every polygon the same way, so that overlaps add up
// under the nonzero rule instead of cancelling.
func (s *shape) clockwise() {
	for _, p := range s.polys {
		var a float64
		for i := range p {
			j := (i + 1) % len(p)
			a += p[i].x*p[j].y - p[j].x*p[i].y
		}
		if a < 0 {
			for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
				p[i], p[j] = p[j], p[i]
			}
		}
	}
}

// glyph returns the shape of r, or false when the font draws it.
func glyph(r rune, w, h, light float64) (shape, bool) {
	s := shape{alpha: 1}
	switch {
	case r >= 0x2500 && r <= 0x257f:
		if !boxDrawing(&s, r, w, h, light) {
			return s, false
		}
	case r >= 0x2580 && r <= 0x259f:
		block(&s, r, w, h)
	case r >= 0xe0b0 && r <= 0xe0b7:
		if !powerline(&s, r, w, h) {
			return s, false
		}
	default:
		return s, false
	}
	s.clockwise()
	return s, true
}

type weight uint8

const (
	none weight = iota
	thin
	heavy
	double
)

// arms of U+2500–U+254B and U+2550–U+256C: up, right, down, left.
var arms = map[rune]string{
	'─': "0101", '━': "0202", '│': "1010", '┃': "2020",
	'┌': "0110", '┍': "0210", '┎': "0120", '┏': "0220",
	'┐': "0011", '┑': "0012", '┒': "0021", '┓': "0022",
	'└': "1100", '┕': "1200", '┖': "2100", '┗': "2200",
	'┘': "1001", '┙': "1002", '┚': "2001", '┛': "2002",
	'├': "1110", '┝': "1210", '┞': "2110", '┟': "1120", '┠': "2120", '┡': "2210", '┢': "1220", '┣': "2220",
	'┤': "1011", '┥': "1012", '┦': "2011", '┧': "1021", '┨': "2021", '┩': "2012", '┪': "1022", '┫': "2022",
	'┬': "0111", '┭': "0112", '┮': "0211", '┯': "0212", '┰': "0121", '┱': "0122", '┲': "0221", '┳': "0222",
	'┴': "1101", '┵': "1102", '┶': "1201", '┷': "1202", '┸': "2101", '┹': "2102", '┺': "2201", '┻': "2202",
	'┼': "1111", '┽': "1112", '┾': "1211", '┿': "1212", '╀': "2111", '╁': "1121", '╂': "2121", '╃': "2112",
	'╄': "2211", '╅': "1122", '╆': "1221", '╇': "2212", '╈': "1222", '╉': "2122", '╊': "2221", '╋': "2222",
	'═': "0303", '║': "3030",
	'╒': "0310", '╓': "0130", '╔': "0330", '╕': "0013", '╖': "0031", '╗': "0033",
	'╘': "1300", '╙': "3100", '╚': "3300", '╛': "1003", '╜': "3001", '╝': "3003",
	'╞': "1310", '╟': "3130", '╠': "3330", '╡': "1013", '╢': "3031", '╣': "3033",
	'╤': "0313", '╥': "0131", '╦': "0333", '╧': "1303", '╨': "3101", '╩': "3303",
	'╪': "1313", '╫': "3131", '╬': "3333",
	'╴': "0001", '╵': "1000", '╶': "0100", '╷': "0010",
	'╸': "0002", '╹': "2000", '╺': "0200", '╻': "0020",
	'╼': "0201", '╽': "1020", '╾': "0102", '╿': "2010",
}

// dashes: horizontal or not, heavy or not, number of dashes.
var dashes = map[rune]struct {
	horizontal, heavy bool
	n                 int
}{
	'┄': {true, false, 3}, '┅': {true, true, 3}, '┆': {false, false, 3}, '┇': {false, true, 3},
	'┈': {true, false, 4}, '┉': {true, true, 4}, '┊': {false, false, 4}, '┋': {false, true, 4},
	'╌': {true, false, 2}, '╍': {true, true, 2}, '╎': {false, false, 2}, '╏': {false, true, 2},
}

// box draws lines of a cell, on a pixel grid so that they stay crisp.
type box struct {
	w, h  float64
	light float64
}

func (b box) thick(wt weight) float64 {
	switch wt {
	case heavy:
		return 2 * b.light
	case double:
		return 3 * b.light
	}
	return b.light
}

// start is where a line of weight wt begins across an axis of length l.
func (b box) start(l float64, wt weight) float64 {
	return math.Floor((l - b.thick(wt)) / 2)
}

// edge is where an arm going toward dir (+1 or -1) along an axis of length l
// starts, against the lines of weight wt crossing it: on the near side of
// a double line when inner.
func (b box) edge(l float64, dir int, wt weight, inner bool) float64 {
	if wt == none {
		return math.Floor(l / 2)
	}
	s := b.start(l, wt)
	if wt == double {
		switch {
		case dir > 0 && inner:
			return s + 2*b.light
		case dir > 0:
			return s
		case inner:
			return s + b.light
		}
		return s + 3*b.light
	}
	if dir > 0 {
		return s
	}
	return s + b.thick(wt)
}

func widest(a, c weight) weight {
	if a == double || c == double {
		return double
	}
	return max(a, c)
}

// arm returns the extents of an arm's lines: along its axis (of length l)
// and across it (of length m). before and after are the arms crossing it,
// on the side of the lower and the higher coordinate.
func (b box) arm(l, m float64, dir int, wt, before, after weight) (spans [][4]float64) {
	along := func(e float64) (float64, float64) {
		if dir > 0 {
			return e, l
		}
		return 0, e
	}
	if wt != double {
		a0, a1 := along(b.edge(l, dir, widest(before, after), false))
		c := b.start(m, wt)
		return [][4]float64{{a0, a1, c, c + b.thick(wt)}}
	}
	c := b.start(m, double)
	for i, same := range []weight{before, after} {
		other := after
		if i == 1 {
			other = before
		}
		e := b.edge(l, dir, other, false)
		if same != none {
			e = b.edge(l, dir, same, true)
		}
		a0, a1 := along(e)
		c0 := c + float64(2*i)*b.light
		spans = append(spans, [4]float64{a0, a1, c0, c0 + b.light})
	}
	return spans
}

func boxDrawing(s *shape, r rune, w, h, light float64) bool {
	b := box{w, h, light}
	if a, ok := arms[r]; ok {
		up, right, down, left := weight(a[0]-'0'), weight(a[1]-'0'), weight(a[2]-'0'), weight(a[3]-'0')
		for _, a := range []struct {
			dir int
			wt  weight
		}{{1, right}, {-1, left}} {
			if a.wt != none {
				for _, sp := range b.arm(w, h, a.dir, a.wt, up, down) {
					s.rect(sp[0], sp[2], sp[1], sp[3])
				}
			}
		}
		for _, a := range []struct {
			dir int
			wt  weight
		}{{1, down}, {-1, up}} {
			if a.wt != none {
				for _, sp := range b.arm(h, w, a.dir, a.wt, left, right) {
					s.rect(sp[2], sp[0], sp[3], sp[1])
				}
			}
		}
		return true
	}
	if d, ok := dashes[r]; ok {
		wt := thin
		if d.heavy {
			wt = heavy
		}
		l, m := w, h
		if !d.horizontal {
			l, m = h, w
		}
		seg := l / float64(d.n)
		gap := math.Max(math.Round(seg/4), 1)
		c := b.start(m, wt)
		for i := range d.n {
			a0 := math.Round(float64(i)*seg + gap/2)
			a1 := math.Round(float64(i+1)*seg - gap/2)
			if d.horizontal {
				s.rect(a0, c, a1, c+b.thick(wt))
			} else {
				s.rect(c, a0, c+b.thick(wt), a1)
			}
		}
		return true
	}
	cx, cy := b.start(w, thin)+light/2, b.start(h, thin)+light/2
	switch r {
	case '╭', '╮', '╯', '╰':
		hx, vy := 1.0, 1.0
		if r == '╮' || r == '╯' {
			hx = -1
		}
		if r == '╯' || r == '╰' {
			vy = -1
		}
		rad := math.Min(math.Min(w-cx, cx), math.Min(h-cy, cy))
		ey, ex := -1.0, -1.0
		if vy > 0 {
			ey = h + 1
		}
		if hx > 0 {
			ex = w + 1
		}
		path := []point{{cx, ey}, {cx, cy + vy*rad}}
		ox, oy := cx+hx*rad, cy+vy*rad
		t1 := 0.0
		if hx > 0 {
			t1 = math.Pi
		}
		t2 := math.Pi / 2
		if vy > 0 {
			t2 = -math.Pi / 2
		}
		d := math.Remainder(t2-t1, 2*math.Pi)
		const steps = 24
		for i := 1; i < steps; i++ {
			t := t1 + d*float64(i)/steps
			path = append(path, point{ox + rad*math.Cos(t), oy + rad*math.Sin(t)})
		}
		path = append(path, point{cx + hx*rad, cy}, point{ex, cy})
		s.stroke(path, light)
		return true
	case '╱', '╲', '╳':
		if r != '╲' {
			s.stroke(extend(point{0, h}, point{w, 0}), light)
		}
		if r != '╱' {
			s.stroke(extend(point{0, 0}, point{w, h}), light)
		}
		return true
	}
	return false
}

// extend lengthens a segment a little on both ends, so that it joins its
// neighbours' once clipped to the cell.
func extend(a, b point) []point {
	dx, dy := unit(b.x-a.x, b.y-a.y)
	return []point{{a.x - dx, a.y - dy}, {b.x + dx, b.y + dy}}
}

func block(s *shape, r rune, w, h float64) {
	ry := func(f float64) float64 { return math.Round(h * f) }
	rx := func(f float64) float64 { return math.Round(w * f) }
	switch {
	case r == '▀':
		s.rect(0, 0, w, ry(0.5))
	case r >= '▁' && r <= '█':
		s.rect(0, h-ry(float64(r-'▀')/8), w, h)
	case r >= '▉' && r <= '▏':
		s.rect(0, 0, rx(float64('▐'-r)/8), h)
	case r == '▐':
		s.rect(rx(0.5), 0, w, h)
	case r >= '░' && r <= '▓':
		s.rect(0, 0, w, h)
		s.alpha = float64(r-'░'+1) / 4
	case r == '▔':
		s.rect(0, 0, w, ry(1.0/8))
	case r == '▕':
		s.rect(w-rx(1.0/8), 0, w, h)
	default:
		// quadrants: upper left, upper right, lower left, lower right
		q := map[rune]string{
			'▖': "0010", '▗': "0001", '▘': "1000", '▙': "1011", '▚': "1001",
			'▛': "1110", '▜': "1101", '▝': "0100", '▞': "0110", '▟': "0111",
		}[r]
		mx, my := rx(0.5), ry(0.5)
		quads := [4][4]float64{{0, 0, mx, my}, {mx, 0, w, my}, {0, my, mx, h}, {mx, my, w, h}}
		for i := range q {
			if q[i] == '1' {
				s.rect(quads[i][0], quads[i][1], quads[i][2], quads[i][3])
			}
		}
	}
}

func powerline(s *shape, r rune, w, h float64) bool {
	switch r {
	case '':
		s.polys = append(s.polys, []point{{0, 0}, {w, h / 2}, {0, h}})
	case '':
		s.polys = append(s.polys, []point{{w, 0}, {0, h / 2}, {w, h}})
	case '', '':
		// half an ellipse, flat side on the cell's edge
		x0, dir := 0.0, 1.0
		if r == '' {
			x0, dir = w, -1
		}
		const steps = 32
		var p []point
		for i := 0; i <= steps; i++ {
			t := -math.Pi/2 + math.Pi*float64(i)/steps
			p = append(p, point{x0 + dir*w*math.Cos(t), h/2 + h/2*math.Sin(t)})
		}
		s.polys = append(s.polys, p)
	default:
		return false
	}
	return true
}
