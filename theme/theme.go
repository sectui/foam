// Package theme is the look shared by foam tools. Apps embed Theme in their
// config with `koanf:",squash"` and call Set once it is loaded.
package theme

import (
	"charm.land/lipgloss/v2"
)

type Theme struct {
	BorderStyle string `koanf:"borderStyle" validate:"oneof=single double rounded heavy"`
	NerdFonts   bool   `koanf:"nerdFonts"`
	Emoji       bool   `koanf:"emoji"`
	// Nerd Fonts only.
	RoundedBadges bool `koanf:"roundedBadges"`
	// An empty line under the top bar and above the help bar.
	Spacious              bool    `koanf:"spacious"`
	HelpBar               bool    `koanf:"helpBar"`
	TransparentBackground bool    `koanf:"transparentBackground"`
	Mouse                 bool    `koanf:"mouse"`
	Colors                Palette `koanf:"colors"`
}

type Palette struct {
	Base00 Color `koanf:"base00"`
	Base01 Color `koanf:"base01"`
	Base02 Color `koanf:"base02"`
	Base03 Color `koanf:"base03"`
	Base04 Color `koanf:"base04"`
	Base05 Color `koanf:"base05"`
	Base06 Color `koanf:"base06"`
	Base07 Color `koanf:"base07"`
	Base08 Color `koanf:"base08"`
	Base09 Color `koanf:"base09"`
	Base0A Color `koanf:"base0A"`
	Base0B Color `koanf:"base0B"`
	Base0C Color `koanf:"base0C"`
	Base0D Color `koanf:"base0D"`
	Base0E Color `koanf:"base0E"`
	Base0F Color `koanf:"base0F"`
	Base10 Color `koanf:"base10"`
	Base11 Color `koanf:"base11"`
	Base12 Color `koanf:"base12"`
	Base13 Color `koanf:"base13"`
	Base14 Color `koanf:"base14"`
	Base15 Color `koanf:"base15"`
	Base16 Color `koanf:"base16"`
	Base17 Color `koanf:"base17"`
}

var current = Theme{
	Mouse: true, NerdFonts: false, HelpBar: true, BorderStyle: "rounded",
	Colors: Palette{
		Base00: "#1e1e2e", Base01: "#181825", Base02: "#313244", Base03: "#45475a",
		Base04: "#585b70", Base05: "#cdd6f4", Base06: "#f5e0dc", Base07: "#b4befe",
		Base08: "#f38ba8", Base09: "#fab387", Base0A: "#f9e2af", Base0B: "#a6e3a1",
		Base0C: "#94e2d5", Base0D: "#89b4fa", Base0E: "#cba6f7", Base0F: "#f2cdcd",
		Base10: "#181825", Base11: "#11111b", Base12: "#eba0ac", Base13: "#f5e0dc",
		Base14: "#a6e3a1", Base15: "#89dceb", Base16: "#74c7ec", Base17: "#f5c2e7",
	},
}

func Set(t Theme) { current = t }

func Current() Theme { return current }

func (t Theme) Border() lipgloss.Border {
	switch t.BorderStyle {
	case "single":
		return lipgloss.NormalBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "heavy":
		return lipgloss.ThickBorder()
	}
	return lipgloss.RoundedBorder()
}

func Mouse() bool             { return current.Mouse }
func Spacious() bool          { return current.Spacious }
func NerdFonts() bool         { return current.NerdFonts }
func HelpBar() bool           { return current.HelpBar }
func Border() lipgloss.Border { return current.Border() }

func Bg() Color        { return current.Colors.Base00 }
func Surface() Color   { return current.Colors.Base01 }
func Selection() Color { return current.Colors.Base02 }
func Muted() Color     { return current.Colors.Base03 }
func Subtle() Color    { return current.Colors.Base04 }
func Fg() Color        { return current.Colors.Base05 }
func Danger() Color    { return current.Colors.Base08 }
func Orange() Color    { return current.Colors.Base09 }
func Warning() Color   { return current.Colors.Base0A }
func Success() Color   { return current.Colors.Base0B }
func Info() Color      { return current.Colors.Base0C }
func Primary() Color   { return current.Colors.Base0D }
func Accent() Color    { return current.Colors.Base0E }
func Brown() Color     { return current.Colors.Base0F }

func Icon(nerd, plain string) string {
	if current.NerdFonts {
		return nerd
	}
	return plain
}
