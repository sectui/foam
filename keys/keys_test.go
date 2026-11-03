package keys

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestCanonical(t *testing.T) {
	for in, want := range map[string]string{
		"q": "q", "Q": "Q", "ctrl+C": "ctrl+c", "Shift+Ctrl+Up": "ctrl+shift+up",
		"escape": "esc", "space": "space", "plus": "+", "alt+comma": "alt+,",
		"shift+q": "Q", "shift+Q": "Q", "ctrl+shift+Q": "ctrl+shift+q", "shift+1": "shift+1",
	} {
		if got, err := Canonical(in); err != nil || got != want {
			t.Errorf("Canonical(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "ctrl+", "hyper+x", "foo"} {
		if _, err := Canonical(bad); err == nil {
			t.Errorf("Canonical(%q) should fail", bad)
		}
	}
}

func TestMatches(t *testing.T) {
	cases := []struct {
		spec string
		msg  tea.KeyPressMsg
		want bool
	}{
		{"ctrl+h", tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl}, true},
		{"h", tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl}, false},
		{"?", tea.KeyPressMsg{Code: '/', Text: "?", Mod: tea.ModShift}, true},
		{"Q", tea.KeyPressMsg{Code: 'q', Text: "Q", Mod: tea.ModShift}, true},
		{"space", tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, true},
		{"enter", tea.KeyPressMsg{Code: tea.KeyEnter}, true},
		{"shift+tab", tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, true},
		{"shift+q", tea.KeyPressMsg{Code: 'Q', Text: "Q"}, true}, // legacy terminal
		{"shift+q", tea.KeyPressMsg{Code: 'q', Text: "Q", Mod: tea.ModShift}, true},
	}
	for _, c := range cases {
		if got := Key(c.spec).Matches(c.msg); got != c.want {
			t.Errorf("%q vs %q (%q): got %v", c.spec, c.msg.String(), c.msg.Keystroke(), got)
		}
	}
}
