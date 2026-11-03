package keys

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

var namedKeys = map[string]string{
	"enter": "enter", "return": "enter",
	"tab": "tab",
	"esc": "esc", "escape": "esc",
	"backspace": "backspace",
	"delete":    "delete", "del": "delete",
	"insert": "insert", "ins": "insert",
	"up": "up", "down": "down", "left": "left", "right": "right",
	"home": "home", "end": "end",
	"pgup": "pgup", "pageup": "pgup",
	"pgdown": "pgdown", "pagedown": "pgdown",
	"space": "space",
	"f1":    "f1", "f2": "f2", "f3": "f3", "f4": "f4", "f5": "f5", "f6": "f6",
	"f7": "f7", "f8": "f8", "f9": "f9", "f10": "f10", "f11": "f11", "f12": "f12",
}

var namedRunes = map[string]string{"plus": "+", "comma": ","}

// Canonical normalizes a spec to bubbletea's format ("ctrl+alt+shift+x").
func Canonical(spec string) (string, error) {
	s := strings.TrimSpace(spec)
	if s == "" {
		return "", fmt.Errorf("empty key")
	}
	parts := strings.Split(s, "+")
	name := strings.TrimSpace(parts[len(parts)-1])
	if name == "" {
		return "", fmt.Errorf("%q: missing key (write \"plus\" for the + key)", spec)
	}
	var ctrl, alt, shift bool
	for _, m := range parts[:len(parts)-1] {
		switch strings.ToLower(strings.TrimSpace(m)) {
		case "ctrl":
			ctrl = true
		case "alt":
			alt = true
		case "shift":
			shift = true
		default:
			return "", fmt.Errorf("%q: unknown modifier %q (ctrl, alt, shift)", spec, m)
		}
	}

	lower := strings.ToLower(name)
	switch {
	case namedKeys[lower] != "":
		name = namedKeys[lower]
	case namedRunes[lower] != "":
		name = namedRunes[lower]
	case len([]rune(name)) == 1:
		switch {
		case ctrl || alt:
			name = lower // terminals report ctrl+a, not ctrl+A
		case shift && strings.ToUpper(lower) != lower:
			name, shift = strings.ToUpper(lower), false // shift+q is "Q" everywhere
		}
	default:
		return "", fmt.Errorf("%q: unknown key %q", spec, name)
	}

	var b strings.Builder
	if ctrl {
		b.WriteString("ctrl+")
	}
	if alt {
		b.WriteString("alt+")
	}
	if shift {
		b.WriteString("shift+")
	}
	b.WriteString(name)
	return b.String(), nil
}

type Key string

func (k *Key) UnmarshalText(b []byte) error {
	s := strings.TrimSpace(string(b))
	if _, err := Canonical(s); err != nil {
		return err
	}
	*k = Key(s)
	return nil
}

func (k Key) Canonical() string {
	c, _ := Canonical(string(k))
	return c
}

// Matches compares against both the text ("?", "Q") and the keystroke ("shift+/").
func (k Key) Matches(msg tea.KeyPressMsg) bool {
	c := k.Canonical()
	return c != "" && (c == msg.String() || c == msg.Keystroke())
}

type List []Key

func (l List) Matches(msg tea.KeyPressMsg) bool {
	for _, k := range l {
		if k.Matches(msg) {
			return true
		}
	}
	return false
}

func (l List) String() string {
	s := make([]string, len(l))
	for i, k := range l {
		s[i] = string(k)
	}
	return strings.Join(s, " / ")
}

// Printable reports keys that type text (no ctrl/alt), stolen by text inputs.
func Printable(msg tea.KeyPressMsg) bool {
	k := msg.Key()
	return k.Text != "" && k.Mod&(tea.ModCtrl|tea.ModAlt) == 0
}
