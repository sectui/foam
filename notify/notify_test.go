package notify

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestCenter(t *testing.T) {
	c := Center{Max: 2}
	for _, s := range []string{"a", "b", "c"} {
		if _, ok := c.Update(Msg{Text: s}); !ok {
			t.Fatal("Msg not handled")
		}
	}
	if h := c.History(); len(h) != 2 || h[0].Text != "b" {
		t.Fatalf("history = %+v", h)
	}
	if e, ok := c.Current(); !ok || e.Text != "c" || c.Unread() != 3 {
		t.Fatalf("current = %+v, unread = %d", e, c.Unread())
	}
	c.Update(expireMsg{id: 3})
	if e, _ := c.Current(); e.Text != "b" {
		t.Fatalf("after expiry, current = %+v", e)
	}
	c.MarkRead()
	if c.Unread() != 0 {
		t.Fatal("unread after MarkRead")
	}
}

func TestToastWraps(t *testing.T) {
	e := Entry{Level: ErrorLevel, Text: "a long message that cannot hold on a single line of this toast"}
	lines := strings.Split(e.Toast(30), "\n")
	if len(lines) < 4 {
		t.Fatalf("not wrapped: %q", lines)
	}
	for _, l := range lines {
		if w := ansi.StringWidth(l); w != 30 {
			t.Fatalf("line %q is %d wide", ansi.Strip(l), w)
		}
	}
	// inside the border, no line may reset its background midway: each
	// reset is followed by a new background before any text
	for _, l := range lines[1 : len(lines)-1] {
		body := ansi.Cut(l, 1, 29)
		for _, part := range strings.Split(body, "\x1b[m")[1:] {
			if part != "" && !strings.HasPrefix(part, "\x1b[") {
				t.Fatalf("text without background after a reset in %q", body)
			}
		}
	}
}
