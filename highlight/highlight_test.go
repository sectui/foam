package highlight

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestWidthsKept(t *testing.T) {
	lines := []string{
		"# steep mode: pitchfork",
		"# a note",
		"POST /FUZZ HTTP/1.1",
		"Host: api.test",
		"Content-Type: application/json",
		"",
		`{"user": "FUZZ", "n": -1.5e3, "ok": true, "x": null}`,
		"<html><a href=\"x\">é</a>",
	}
	out := Message(lines, Options{Keywords: []string{"FUZZ"}})
	for i, l := range out {
		if ansi.Strip(l) != lines[i] {
			t.Errorf("line %d: %q became %q", i, lines[i], ansi.Strip(l))
		}
	}
}

func TestRoles(t *testing.T) {
	l := &line{text: []rune(`{"a": "b", "n": 12}`)}
	l.roles = make([]Role, len(l.text))
	json(l)
	if l.roles[1] != Key || l.roles[6] != Str || l.roles[16] != Number {
		t.Fatalf("roles = %v", l.roles)
	}
	r := &line{text: []rune("HTTP/1.1 404 Not Found")}
	r.roles = make([]Role, len(r.text))
	firstLine(r)
	if r.code != 404 || r.roles[9] != Status || r.roles[0] != Version {
		t.Fatalf("status line: code %d, roles %v", r.code, r.roles)
	}
	if !strings.Contains(MethodColor("GET").Render("x"), "x") {
		t.Fatal("method color")
	}
}

func TestCacheSameAsFresh(t *testing.T) {
	lines := []string{"# steep mode: x", "GET /a HTTP/1.1", "Content-Type: text/html", "", "{not json}", "<b>x</b>"}
	var c Cache
	c.Message(lines, Options{Directive: "steep"})
	// the same lines in other places of the message are colored for there
	moved := []string{"GET /a HTTP/1.1", "# steep mode: x", "", "<b>x</b>", "{not json}"}
	got := c.Message(moved, Options{Directive: "steep"})
	want := Message(moved, Options{Directive: "steep"})
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: cached %q, fresh %q", i, got[i], want[i])
		}
	}
}

func TestURL(t *testing.T) {
	u := "https://api.test:8443/a/file{$w/x}?q=1&r=a=b"
	spans := func(s string) []Span {
		i := strings.Index(s, "file{")
		return []Span{{Start: i, End: i + len("file{$w/x}"), Role: Block}}
	}
	got := URL(u, spans)
	if ansi.Strip(got) != u {
		t.Fatalf("text changed: %q", ansi.Strip(got))
	}
	for _, part := range []string{
		style(Target, 0).Render("api.test:8443"),
		style(Path, 0).Render("/a/"),
		style(Block, 0).Render("file{$w/x}"),
		style(Key, 0).Render("q"),
		style(Str, 0).Render("a=b"),
	} {
		if !strings.Contains(got, part) {
			t.Errorf("%q not colored as expected in %q", ansi.Strip(part), got)
		}
	}
}

func TestSpansAndDirective(t *testing.T) {
	spans := func(s string) []Span {
		if i := strings.Index(s, "§"); i >= 0 {
			i = len([]rune(s[:i]))
			return []Span{{Start: i, End: i + 1, Role: Keyword}}
		}
		return nil
	}
	lines := []string{"# steep mode: x", "# §", "GET /§ HTTP/1.1"}
	out := Message(lines, Options{Directive: "steep", Spans: spans})
	if !strings.Contains(out[0], style(DirectiveKey, 0).Render("mode")) {
		t.Errorf("directive key not colored: %q", out[0])
	}
	if strings.Contains(out[1], style(Keyword, 0).Render("§")) {
		t.Errorf("spans in a comment: %q", out[1])
	}
	if !strings.Contains(out[2], style(Keyword, 0).Render("§")) {
		t.Errorf("span not colored: %q", out[2])
	}
	if plain := Message(lines[:1], Options{}); strings.Contains(plain[0], style(DirectiveKey, 0).Render("mode")) {
		t.Errorf("directive colored without Directive: %q", plain[0])
	}
}
