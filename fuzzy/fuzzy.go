// Package fuzzy matches a query against text the way fzf does: the letters of
// the query in order, scored so that contiguous runs and word starts win.
package fuzzy

import (
	"slices"
	"strings"
	"unicode"
)

// Match reports whether every rune of query appears in s in order, ignoring
// case. A higher score is a better match; positions are rune indexes in s.
func Match(query, s string) (score int, positions []int, ok bool) {
	q := []rune(strings.ToLower(query))
	if len(q) == 0 {
		return 0, nil, true
	}
	r := []rune(s)
	lower := []rune(strings.ToLower(s))
	if len(lower) != len(r) {
		lower = r // a case mapping that changes length: match as is
	}

	// A plain substring is always the best way to match.
	if i := strings.Index(string(lower), string(q)); i >= 0 {
		start := len([]rune(string(lower)[:i]))
		for k := range q {
			positions = append(positions, start+k)
		}
		return 1000 + bonus(r, start)*10 - start, positions, true
	}

	// Greedy forward pass, then a backward pass to tighten the window.
	qi := 0
	end := -1
	for i, c := range lower {
		if c == q[qi] {
			qi++
			if qi == len(q) {
				end = i
				break
			}
		}
	}
	if end < 0 {
		return 0, nil, false
	}
	qi = len(q) - 1
	for i := end; i >= 0 && qi >= 0; i-- {
		if lower[i] == q[qi] {
			positions = append(positions, i)
			qi--
		}
	}
	slices.Reverse(positions)

	prev := -2
	for _, p := range positions {
		score += 10 + bonus(r, p)*5
		if p == prev+1 {
			score += 15
		}
		prev = p
	}
	score -= positions[len(positions)-1] - positions[0]
	return score, positions, true
}

func bonus(r []rune, i int) int {
	if i == 0 {
		return 3
	}
	p := r[i-1]
	switch {
	case p == '/' || p == '\\':
		return 3
	case p == '.' || p == '-' || p == '_' || unicode.IsSpace(p):
		return 2
	case unicode.IsLower(p) && unicode.IsUpper(r[i]):
		return 1
	}
	return 0
}

// Filter returns the indexes of items matching query, best first; ties keep
// the order of items.
func Filter(query string, items []string) []int {
	type hit struct{ i, score int }
	var hits []hit
	for i, s := range items {
		if sc, _, ok := Match(query, s); ok {
			hits = append(hits, hit{i, sc})
		}
	}
	slices.SortStableFunc(hits, func(a, b hit) int { return b.score - a.score })
	out := make([]int, len(hits))
	for i, h := range hits {
		out[i] = h.i
	}
	return out
}
