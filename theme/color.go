package theme

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// Color is a "#rrggbb" string, validated when decoded from config.
type Color string

func parseHex(s string) (r, g, b uint8, err error) {
	h, ok := strings.CutPrefix(s, "#")
	if !ok || len(h) != 6 {
		return 0, 0, 0, fmt.Errorf("%q is not a #rrggbb color", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%q is not a #rrggbb color", s)
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v), nil
}

func (c *Color) UnmarshalText(b []byte) error {
	s := strings.TrimSpace(string(b))
	if _, _, _, err := parseHex(s); err != nil {
		return err
	}
	*c = Color(s)
	return nil
}

func (c Color) Value() color.Color {
	if _, _, _, err := parseHex(string(c)); err != nil {
		return lipgloss.NoColor{}
	}
	return lipgloss.Color(string(c))
}

func (c Color) Style() lipgloss.Style { return lipgloss.NewStyle().Foreground(c.Value()) }

func (c Color) Render(s string) string { return c.Style().Render(s) }

// Mix blends c toward o: 0 is c, 1 is o.
func (c Color) Mix(o Color, ratio float64) Color {
	r1, g1, b1, _ := parseHex(string(c))
	r2, g2, b2, _ := parseHex(string(o))
	mix := func(a, b uint8) uint8 { return uint8(float64(a)*(1-ratio) + float64(b)*ratio + 0.5) }
	return Color(fmt.Sprintf("#%02x%02x%02x", mix(r1, r2), mix(g1, g2), mix(b1, b2)))
}

func (c Color) Dim() Color { return c.Mix(Bg(), 0.7) }
