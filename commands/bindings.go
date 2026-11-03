package commands

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/sectui/foam/keys"
)

// Bindings is the user's `keys:` section, nested by command ID:
//
//	keys:
//	  app:
//	    quit: [q, ctrl+c]
//	  collection:
//	    delete: d     # or "d,delete" from env / --set
//	    rename: []    # unbound
type Bindings map[string]any

func (b Bindings) Validate() error {
	_, err := b.Flatten()
	return err
}

func (b Bindings) Flatten() (map[string]keys.List, error) {
	out := map[string]keys.List{}
	var errs []error
	add := func(id string, specs []string) {
		l := keys.List{}
		for _, s := range specs {
			if s = strings.TrimSpace(s); s == "" {
				continue
			}
			if _, err := keys.Canonical(s); err != nil {
				errs = append(errs, fmt.Errorf("keys.%s: %w", id, err))
				continue
			}
			l = append(l, keys.Key(s))
		}
		out[id] = l
	}
	var walk func(prefix string, v any)
	walk = func(prefix string, v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, vv := range x {
				id := k
				if prefix != "" {
					id = prefix + "." + k
				}
				walk(id, vv)
			}
		case nil:
			add(prefix, nil)
		case string:
			add(prefix, strings.Split(x, ",")) // "," itself is spelled "comma"
		case int, int64, float64:
			add(prefix, []string{fmt.Sprint(x)}) // YAML reads `list: 1` as a number
		case []any:
			specs := make([]string, 0, len(x))
			for _, e := range x {
				switch e.(type) {
				case string, int, int64, float64:
					specs = append(specs, fmt.Sprint(e))
				default:
					errs = append(errs, fmt.Errorf("keys.%s: %v is not a key", prefix, e))
				}
			}
			add(prefix, specs)
		case []string:
			add(prefix, x)
		default:
			errs = append(errs, fmt.Errorf("keys.%s: invalid value %v", prefix, v))
		}
	}
	walk("", map[string]any(b))
	slices.SortFunc(errs, func(a, b error) int { return strings.Compare(a.Error(), b.Error()) })
	return out, errors.Join(errs...)
}
