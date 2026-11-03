// Package geom is rects, size distribution and neighbors.
package geom

type Rect struct{ X, Y, W, H int }

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }

func (r Rect) Inset(n int) Rect {
	return Rect{r.X + n, r.Y + n, max(r.W-2*n, 0), max(r.H-2*n, 0)}
}

// Size is Fixed (cells) or Ratio (share of what is left).
type Size struct {
	Fixed int
	Ratio int
}

func Fixed(n int) Size { return Size{Fixed: n} }
func Ratio(n int) Size { return Size{Ratio: n} }

// Fixed sizes first, then the rest by ratio.
func Distribute(total int, sizes []Size) []int {
	out := make([]int, len(sizes))
	left, ratios := total, 0
	for i, s := range sizes {
		if s.Fixed > 0 {
			out[i] = min(s.Fixed, max(left, 0))
			left -= out[i]
		} else {
			ratios += max(s.Ratio, 1)
		}
	}
	if left <= 0 || len(out) == 0 {
		return out
	}
	if ratios == 0 {
		out[len(out)-1] += left // no hole when only fixed sizes are left
		return out
	}
	given, last := 0, -1
	for i, s := range sizes {
		if s.Fixed > 0 {
			continue
		}
		out[i] = left * max(s.Ratio, 1) / ratios
		given += out[i]
		last = i
	}
	out[last] += left - given // rounding goes to the last one
	return out
}

type Dir int

const (
	Left Dir = iota
	Down
	Up
	Right
)

// Neighbor returns the index of the closest rect in dir from `from`, or -1.
// Overlapping rects win, then the closest, then the best edge-aligned.
func Neighbor(from Rect, cands []Rect, dir Dir) int {
	best, bestScore := -1, 0
	for i, c := range cands {
		var gap, overlap, offset int
		switch dir {
		case Left:
			gap = from.X - (c.X + c.W)
		case Right:
			gap = c.X - (from.X + from.W)
		case Up:
			gap = from.Y - (c.Y + c.H)
		case Down:
			gap = c.Y - (from.Y + from.H)
		}
		if gap < 0 {
			continue
		}
		if dir == Left || dir == Right {
			overlap = min(from.Y+from.H, c.Y+c.H) - max(from.Y, c.Y)
			offset = abs(c.Y - from.Y)
		} else {
			overlap = min(from.X+from.W, c.X+c.W) - max(from.X, c.X)
			offset = abs(c.X - from.X)
		}
		score := gap*1000 + offset
		if overlap <= 0 {
			score += 1_000_000
		}
		if best < 0 || score < bestScore {
			best, bestScore = i, score
		}
	}
	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (r Rect) Columns(sizes ...Size) []Rect {
	out := make([]Rect, len(sizes))
	x := r.X
	for i, w := range Distribute(r.W, sizes) {
		out[i] = Rect{x, r.Y, w, r.H}
		x += w
	}
	return out
}

func (r Rect) Rows(sizes ...Size) []Rect {
	out := make([]Rect, len(sizes))
	y := r.Y
	for i, h := range Distribute(r.H, sizes) {
		out[i] = Rect{r.X, y, r.W, h}
		y += h
	}
	return out
}

func (r Rect) Local(x, y int) (int, int) { return x - r.X, y - r.Y }
