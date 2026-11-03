package shot

import (
	"bytes"
	"image/png"
	"os/exec"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// bounds is the box around a shape's polygons.
func bounds(s shape) (x0, y0, x1, y1 float64) {
	x0, y0 = 1e9, 1e9
	for _, p := range s.polys {
		for _, q := range p {
			x0, y0, x1, y1 = min(x0, q.x), min(y0, q.y), max(x1, q.x), max(y1, q.y)
		}
	}
	return
}

func TestBoxDrawing(t *testing.T) {
	const w, h = 10, 20
	for _, tc := range []struct {
		r              rune
		x0, y0, x1, y1 float64
	}{
		{'─', 0, 9, 10, 10},
		{'│', 4, 0, 5, 20},
		{'┌', 4, 9, 10, 20},
		{'┘', 0, 0, 5, 10},
		{'━', 0, 9, 10, 11},
		{'═', 0, 8, 10, 11},
		{'╔', 3, 8, 10, 20},
		{'▀', 0, 0, 10, 10},
		{'▐', 5, 0, 10, 20},
	} {
		s, ok := glyph(tc.r, w, h, 1)
		if !ok {
			t.Fatalf("%c: not drawn", tc.r)
		}
		x0, y0, x1, y1 := bounds(s)
		if x0 != tc.x0 || y0 != tc.y0 || x1 != tc.x1 || y1 != tc.y1 {
			t.Errorf("%c: bounds (%g,%g)-(%g,%g), want (%g,%g)-(%g,%g)", tc.r, x0, y0, x1, y1, tc.x0, tc.y0, tc.x1, tc.y1)
		}
	}
	if _, ok := glyph('a', w, h, 1); ok {
		t.Error("a letter is left to the font")
	}
	if s, _ := glyph('▒', w, h, 1); s.alpha != 0.5 {
		t.Errorf("▒ alpha = %g", s.alpha)
	}
}

func canvas(s string) *lipgloss.Canvas {
	c := lipgloss.NewCanvas(lipgloss.Width(s), lipgloss.Height(s))
	c.Compose(lipgloss.NewLayer(s))
	return c
}

func TestSVG(t *testing.T) {
	c := canvas(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff0000")).Render("<a&b>") + " ╭─╮")
	var b bytes.Buffer
	if err := SVG(&b, c, Options{Font: "/nonexistent/Font.ttf"}); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	for _, want := range []string{`&lt;a&amp;b&gt;`, `fill="#ff0000" class="b"`, `font-family:'Font', monospace`, `<path d="M`} {
		if !strings.Contains(s, want) {
			t.Errorf("no %q in\n%s", want, s)
		}
	}
	if strings.Contains(s, "╭") {
		t.Error("box drawing left as text")
	}
}

func TestPNG(t *testing.T) {
	if _, err := exec.LookPath("fc-match"); err != nil {
		t.Skip("no fontconfig")
	}
	c := canvas("ab\n╰─")
	var b bytes.Buffer
	if err := PNG(&b, c, Options{Scale: 2}); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(&b)
	if err != nil {
		t.Fatal(err)
	}
	if r := img.Bounds(); r.Dx()%2 != 0 || r.Dy()%2 != 0 || r.Dx() == 0 {
		t.Fatalf("bounds %v", r)
	}
}

func TestMissingFontFile(t *testing.T) {
	if err := PNG(&bytes.Buffer{}, canvas("a"), Options{Font: "/nonexistent/Font.ttf"}); err == nil {
		t.Fatal("no error")
	}
}
