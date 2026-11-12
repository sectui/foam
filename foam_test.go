package foam_test

import (
	"testing"

	"github.com/anotherhadi/foam"
	"github.com/anotherhadi/foam/commands"
	"github.com/anotherhadi/foam/config"
	"github.com/anotherhadi/foam/confirm"
	"github.com/anotherhadi/foam/form"
	"github.com/anotherhadi/foam/help"
	"github.com/anotherhadi/foam/list"
	"github.com/anotherhadi/foam/palette"
	"github.com/anotherhadi/foam/prompt"
	"github.com/anotherhadi/foam/theme"
	"github.com/anotherhadi/foam/tree"
	"github.com/anotherhadi/foam/viewer"
)

type cfg struct {
	Theme theme.Theme       `koanf:",squash"`
	Keys  commands.Bindings `koanf:"keys"`
}

type noSource struct{}

func (noSource) Fields() []form.Field     { return nil }
func (noSource) Set(string, string) error { return nil }
func (noSource) Reset(string) error       { return nil }

// Every command of a foam component has keys in Defaults.
func TestDefaultsBindEveryCommand(t *testing.T) {
	c, err := config.LoadBytes[cfg](foam.Defaults)
	if err != nil {
		t.Fatal(err)
	}
	reg, err := commands.New(c.Keys)
	if err != nil {
		t.Fatal(err)
	}
	var l list.Model
	var v viewer.Model
	f := form.New("f", noSource{})
	tr := tree.New(func(s string) (string, bool) { return s, false })
	scopes := []commands.Scope{
		{Name: "list", Commands: l.Commands()},
		{Name: "viewer", Commands: v.Commands()},
		help.New(nil).Scope(),
		palette.New(nil).Scope(),
		confirm.New("t", "q", nil).Scope(),
		prompt.New("t", "", nil).Scope(),
		f.Scope(),
		{Name: "tree", Commands: tr.Commands()},
	}
	if err := reg.Check(scopes); err != nil {
		t.Fatal(err)
	}
}
