package commands

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/anotherhadi/foam/keys"
)

type Command struct {
	ID       string      // "collection.delete": config key and Run target
	Title    string      // palette; empty = not in the palette
	Help     string      // help bar label; empty = not in the help bar
	Optional bool        // may be absent from Bindings, then it has no key
	When     func() bool // nil = always
	Run      func() tea.Cmd
}

func (c Command) Enabled() bool { return c.Run != nil && (c.When == nil || c.When()) }

// Scope is one level of the focus path (app, page, pane, overlay).
type Scope struct {
	Name     string
	Commands []Command
}

type Binding struct {
	Command
	Scope string
	Keys  keys.List // effective keys, after overrides and shadowing
}

type Registry struct {
	bindings map[string]keys.List
}

func New(b Bindings) (*Registry, error) {
	o, err := b.Flatten()
	if err != nil {
		return nil, err
	}
	return &Registry{bindings: o}, nil
}

func (r *Registry) KeysOf(c Command) keys.List { return r.bindings[c.ID] }

// Active lists enabled commands, deepest scope first; deeper keys shadow outer ones.
func (r *Registry) Active(path []Scope) []Binding {
	taken := map[string]bool{}
	var out []Binding
	for i := len(path) - 1; i >= 0; i-- {
		for _, c := range path[i].Commands {
			if !c.Enabled() {
				continue
			}
			b := Binding{Command: c, Scope: path[i].Name}
			for _, k := range r.KeysOf(c) {
				if canon := k.Canonical(); !taken[canon] {
					taken[canon] = true
					b.Keys = append(b.Keys, k)
				}
			}
			out = append(out, b)
		}
	}
	return out
}

func (r *Registry) Match(path []Scope, msg tea.KeyPressMsg) (Binding, bool) {
	for _, b := range r.Active(path) {
		if b.Keys.Matches(msg) {
			return b, true
		}
	}
	return Binding{}, false
}

func (r *Registry) Find(path []Scope, id string) (Binding, bool) {
	for _, b := range r.Active(path) {
		if b.ID == id {
			return b, true
		}
	}
	return Binding{}, false
}

// Check reports commands without a binding, IDs defined twice and keys bound
// twice in one scope. Bindings of commands it does not know are allowed: they
// may belong to overlays or panes that appear later.
func (r *Registry) Check(scopes []Scope) error {
	var errs []error
	known := map[string]bool{} // ID → optional
	for _, s := range scopes {
		owner := map[string]string{}
		seen := map[string]bool{}
		for _, c := range s.Commands {
			if seen[c.ID] {
				errs = append(errs, fmt.Errorf("command %s defined twice", c.ID))
			}
			seen[c.ID] = true
			opt, ok := known[c.ID]
			known[c.ID] = c.Optional && (!ok || opt)
			for _, k := range r.KeysOf(c) {
				canon := k.Canonical() // already validated by Flatten
				if prev, dup := owner[canon]; dup && prev != c.ID {
					errs = append(errs, fmt.Errorf("keys.%s: %q already used by %s", c.ID, string(k), prev))
					continue
				}
				owner[canon] = c.ID
			}
		}
	}
	var missing []string
	for id, optional := range known {
		if _, ok := r.bindings[id]; !ok && !optional {
			missing = append(missing, "keys."+id)
		}
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		errs = append(errs, fmt.Errorf("no keys for (use [] for none): %s", strings.Join(missing, ", ")))
	}
	return errors.Join(errs...)
}

type Hint struct {
	Keys  string
	Label string
}

func (r *Registry) Hints(path []Scope) []Hint {
	var out []Hint
	for _, b := range r.Active(path) {
		if b.Help != "" && len(b.Keys) > 0 {
			out = append(out, Hint{Keys: b.Keys.String(), Label: b.Help})
		}
	}
	return out
}

type Group struct {
	Scope    string
	Bindings []Binding
}

// Groups splits Active by scope, deepest first.
func (r *Registry) Groups(path []Scope) []Group {
	var out []Group
	for _, b := range r.Active(path) {
		if n := len(out); n > 0 && out[n-1].Scope == b.Scope {
			out[n-1].Bindings = append(out[n-1].Bindings, b)
			continue
		}
		out = append(out, Group{Scope: b.Scope, Bindings: []Binding{b}})
	}
	return out
}
