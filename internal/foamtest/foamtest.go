// Package foamtest holds helpers for the tests of foam's overlays.
package foamtest

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/commands"
)

// Sizes are the boxes overlays are drawn in, tiny ones included.
var Sizes = [][2]int{{0, 0}, {1, 1}, {4, 2}, {10, 3}, {20, 6}, {40, 10}, {80, 24}, {200, 60}}

// Fits fails when view(w, h) draws a line wider than w (or 2, the smallest
// frame), or more lines than h when h is at least minH.
func Fits(t *testing.T, name string, minH int, view func(w, h int) string) {
	t.Helper()
	for _, s := range Sizes {
		w, h := s[0], s[1]
		lines := strings.Split(view(w, h), "\n")
		for i, l := range lines {
			if lw := ansi.StringWidth(l); lw > max(w, 2) {
				t.Errorf("%s at %dx%d: line %d is %d cells wide: %q", name, w, h, i, lw, ansi.Strip(l))
			}
		}
		if h >= minH && len(lines) > h {
			t.Errorf("%s at %dx%d: %d lines", name, w, h, len(lines))
		}
	}
}

// Msgs runs cmd and the batches and sequences it returns.
func Msgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if b, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range b {
			out = append(out, Msgs(c)...)
		}
		return out
	}
	if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeFor[tea.Cmd]() {
		var out []tea.Msg
		for i := range v.Len() {
			out = append(out, Msgs(v.Index(i).Interface().(tea.Cmd))...)
		}
		return out
	}
	if msg == nil {
		return nil
	}
	return []tea.Msg{msg}
}

// Run runs the command id of scope, and returns what it posted.
func Run(t *testing.T, scope commands.Scope, id string) []tea.Msg {
	t.Helper()
	for _, c := range scope.Commands {
		if c.ID == id {
			return Msgs(c.Run())
		}
	}
	t.Fatalf("no command %s in %s", id, scope.Name)
	return nil
}

// Has reports whether one of msgs is a T.
func Has[T any](msgs []tea.Msg) bool {
	for _, m := range msgs {
		if _, ok := m.(T); ok {
			return true
		}
	}
	return false
}

// Type sends s as key presses to update.
func Type(update func(tea.Msg) tea.Cmd, s string) {
	for _, r := range s {
		update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}
