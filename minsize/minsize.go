// Package minsize keeps an app from drawing in a terminal too small for it:
// below its minimum, the app shows what it needs instead.
package minsize

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/theme"
)

type Size struct {
	W, H int
}

func (s Size) Fits(w, h int) bool { return w >= s.W && h >= s.H }

// View fills w×h with what the app needs, the dimension that falls short
// colored.
func (s Size) View(w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	dim := func(have, need int) string {
		c := theme.Success()
		if have < need {
			c = theme.Danger()
		}
		return c.Style().Bold(true).Render(fmt.Sprint(have))
	}
	lines := []string{
		theme.Fg().Style().Bold(true).Render("Terminal too small"),
		dim(w, s.W) + theme.Subtle().Render(" × ") + dim(h, s.H),
		theme.Subtle().Render(fmt.Sprintf("needs %d × %d", s.W, s.H)),
	}
	out := make([]string, h)
	top := max((h-len(lines))/2, 0)
	for i := range out {
		l := ""
		if k := i - top; k >= 0 && k < len(lines) {
			l = lines[k]
		}
		l = ansi.Truncate(l, w, "")
		pad := max(w-ansi.StringWidth(l), 0)
		out[i] = strings.Repeat(" ", pad/2) + l + strings.Repeat(" ", pad-pad/2)
	}
	return lipgloss.JoinVertical(lipgloss.Left, out...)
}
