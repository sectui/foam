package fuzzy

import (
	"slices"
	"testing"
)

func TestMatch(t *testing.T) {
	for _, c := range []struct {
		q, s string
		ok   bool
	}{
		{"apitok", "api/v2/token", true},
		{"API", "api/v2/token", true},
		{"tokapi", "api/v2/token", false},
		{"", "anything", true},
		{"v2/to", "api/v2/token", true},
	} {
		if _, _, ok := Match(c.q, c.s); ok != c.ok {
			t.Errorf("Match(%q, %q) = %v", c.q, c.s, ok)
		}
	}
}

func TestPositions(t *testing.T) {
	_, pos, _ := Match("apitok", "api/v2/token")
	if !slices.Equal(pos, []int{0, 1, 2, 7, 8, 9}) {
		t.Fatalf("positions = %v", pos)
	}
}

func TestFilterOrder(t *testing.T) {
	got := Filter("log", []string{"blog/a", "auth/login", "x/l/o/g"})
	if !slices.Equal(got, []int{1, 0, 2}) {
		t.Fatalf("got %v", got)
	}
}
