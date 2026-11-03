package theme

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func TestColor(t *testing.T) {
	var c Color
	if err := c.UnmarshalText([]byte(" #a0b0c0 ")); err != nil || c != "#a0b0c0" {
		t.Fatalf("got %q, %v", c, err)
	}
	for _, bad := range []string{"", "a0b0c0", "#abc", "#gggggg", "#a0b0c0d0"} {
		if err := c.UnmarshalText([]byte(bad)); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if _, ok := Color("nope").Value().(lipgloss.NoColor); !ok {
		t.Error("an invalid color is drawn")
	}
	for _, m := range []struct {
		ratio float64
		want  Color
	}{{0, "#000000"}, {1, "#ffffff"}, {0.5, "#808080"}} {
		if got := Color("#000000").Mix("#ffffff", m.ratio); got != m.want {
			t.Errorf("Mix(%v) = %s, want %s", m.ratio, got, m.want)
		}
	}
}

func TestSet(t *testing.T) {
	old := Current()
	t.Cleanup(func() { Set(old) })
	th := old
	th.NerdFonts, th.BorderStyle = true, "double"
	Set(th)
	if Icon("N", "p") != "N" || Border() != lipgloss.DoubleBorder() {
		t.Error("theme not applied")
	}
	th.NerdFonts, th.BorderStyle = false, "unknown"
	Set(th)
	if Icon("N", "p") != "p" || Border() != lipgloss.RoundedBorder() {
		t.Error("fallbacks")
	}
	for _, s := range []string{"single", "heavy"} {
		th.BorderStyle = s
		if th.Border() == lipgloss.RoundedBorder() {
			t.Errorf("%s border", s)
		}
	}
}
