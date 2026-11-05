// Package config loads a layered configuration into a typed struct.
//
// Layers, later wins: Base (embedded YAML, lowest first) < Files (lowest first,
// missing ones skipped) < Extra files < environment < flags < --set.
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
)

// ErrExit is returned when the command line asked for something that ends the
// program (--help, --print-config): the caller exits with status 0.
var ErrExit = errors.New("exit requested")

type Options struct {
	// Name is used in --help and for EnvPrefix.
	Name  string
	Usage string

	// Embedded YAML, lowest first.
	Base [][]byte
	// Optional, lowest first. "~/" is expanded.
	Files []string
	// Extra adds files named by the merged config, e.g. a project's own. They
	// go above Files and under the environment.
	Extra func(get func(key string) string) []string

	// EnvPrefix defaults to the upper-cased Name followed by "_".
	EnvPrefix string

	Flags func(fs *flag.FlagSet)
	// FlagKeys maps flag names to config keys. The other flags are left to
	// the app.
	FlagKeys map[string]string

	Stdout io.Writer
}

type Result struct {
	Flags *flag.FlagSet
	// Lowest first.
	Loaded []string
}

// Validator is run on the config after decoding.
type Validator interface {
	Validate() error
}

func Load[T any](opts Options, args []string) (*T, Result, error) {
	if opts.Name == "" {
		return nil, Result{}, errors.New("config: Options.Name required")
	}
	if opts.EnvPrefix == "" {
		opts.EnvPrefix = envPrefix(opts.Name)
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}

	fs := flag.NewFlagSet(opts.Name, flag.ContinueOnError)
	fs.SetOutput(opts.Stdout)
	sets := fs.StringArrayP("set", "c", nil, "set a config key: -c key=value (repeatable, comma-separated lists)")
	printCfg := fs.Bool("print-config", false, "print the merged config, then exit")
	if opts.Flags != nil {
		opts.Flags(fs)
	}
	fs.Usage = func() { usage(fs, opts) }
	res := Result{Flags: fs}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, res, ErrExit
		}
		return nil, res, err
	}
	if fs.NArg() > 0 {
		return nil, res, fmt.Errorf("unexpected argument: %q", fs.Arg(0))
	}

	t := reflect.TypeFor[T]()
	l := loader{opts: opts, fs: fs, sets: *sets, lists: fieldPaths(t, reflect.Slice),
		maps: fieldPaths(t, reflect.Map), leaves: fieldPaths(t, 0)}

	files := expandAll(opts.Files)
	k, loaded, err := l.load(files)
	if err != nil {
		return nil, res, err
	}
	if opts.Extra != nil {
		if extra := expandAll(opts.Extra(k.String)); len(extra) > 0 {
			if k, loaded, err = l.load(append(files, extra...)); err != nil {
				return nil, res, err
			}
		}
	}
	res.Loaded = loaded

	cfg := new(T)
	if err := unmarshalStrict(k, cfg); err != nil {
		return nil, res, fmt.Errorf("invalid config:\n%w", err)
	}
	if err := validate(cfg); err != nil {
		return nil, res, fmt.Errorf("invalid config:\n%w", err)
	}

	if *printCfg {
		b, err := k.Marshal(yaml.Parser())
		if err != nil {
			return nil, res, err
		}
		fmt.Fprintf(opts.Stdout, "# files: %s\n%s", strings.Join(loaded, ", "), b)
		return nil, res, ErrExit
	}
	return cfg, res, nil
}

// parser reads YAML with dotted keys spelled either way: `app.quit: q` is
// `app: {quit: q}`, so that both forms override each other.
type parser struct{ yaml *yaml.YAML }

func (p parser) Unmarshal(b []byte) (map[string]any, error) {
	m, err := p.yaml.Unmarshal(b)
	if err != nil {
		return nil, err
	}
	flat, _ := maps.Flatten(m, nil, ".")
	return maps.Unflatten(flat, "."), nil
}

func (p parser) Marshal(m map[string]any) ([]byte, error) { return p.yaml.Marshal(m) }

func yamlParser() parser { return parser{yaml.Parser()} }

type loader struct {
	opts   Options
	fs     *flag.FlagSet
	sets   []string
	lists  map[string]bool
	maps   map[string]bool
	leaves map[string]bool
}

func (l loader) load(files []string) (*koanf.Koanf, []string, error) {
	k := koanf.New(".")
	for _, b := range l.opts.Base {
		if err := k.Load(rawbytes.Provider(b), yamlParser()); err != nil {
			return nil, nil, fmt.Errorf("invalid default config: %w", err)
		}
	}

	var loaded []string
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			continue
		}
		if err := k.Load(file.Provider(f), yamlParser()); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", f, err)
		}
		loaded = append(loaded, f)
	}

	prefix := l.opts.EnvPrefix
	if err := k.Load(env.Provider(".", env.Opt{
		Prefix: prefix,
		TransformFunc: func(key, val string) (string, any) {
			key, ok := envKey(l.leaves, strings.ReplaceAll(strings.TrimPrefix(key, prefix), "__", "."))
			if !ok && !l.underMap(key) {
				return "", nil // not a key: another program's variable, or a typo
			}
			return key, parseValue(l.lists, key, val)
		},
	}), nil); err != nil {
		return nil, nil, err
	}

	if err := k.Load(posflag.ProviderWithFlag(l.fs, ".", k, func(f *flag.Flag) (string, any) {
		key, ok := l.opts.FlagKeys[f.Name]
		if !ok || !f.Changed {
			return "", nil
		}
		if f.Value.Type() == "duration" {
			return key, f.Value.String()
		}
		return key, posflag.FlagVal(l.fs, f)
	}), nil); err != nil {
		return nil, nil, err
	}

	for _, s := range l.sets {
		key, val, ok := strings.Cut(s, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, nil, fmt.Errorf("-c %q: expected key=value", s)
		}
		if !k.Exists(key) && !l.underMap(key) {
			return nil, nil, fmt.Errorf("-c %q: unknown key %q", s, key)
		}
		if _, isMap := k.Get(key).(map[string]any); isMap {
			return nil, nil, fmt.Errorf("-c %q: %q is a section, name a key in it", s, key)
		}
		if err := k.Set(key, parseValue(l.lists, key, val)); err != nil {
			return nil, nil, err
		}
	}
	return k, loaded, nil
}

func (l loader) underMap(key string) bool {
	for p := range l.maps {
		if strings.HasPrefix(key, p+".") {
			return true
		}
	}
	return false
}

// LoadBytes decodes YAML layers, lowest first: for configs that are not the
// program's own (a theme file, a test).
func LoadBytes[T any](layers ...[]byte) (*T, error) {
	k := koanf.New(".")
	for _, b := range layers {
		if err := k.Load(rawbytes.Provider(b), yamlParser()); err != nil {
			return nil, err
		}
	}
	cfg := new(T)
	if err := unmarshalStrict(k, cfg); err != nil {
		return nil, err
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseValue(lists map[string]bool, key, val string) any {
	if !lists[key] {
		return val
	}
	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// fieldPaths lists the koanf paths of the fields of the given kind ([]byte
// excepted): lists are split on commas, maps accept new sub-keys. Kind 0
// lists every leaf.
func fieldPaths(t reflect.Type, kind reflect.Kind) map[string]bool {
	paths := map[string]bool{}
	var walk func(t reflect.Type, prefix string)
	walk = func(t reflect.Type, prefix string) {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return
		}
		for i := range t.NumField() {
			f := t.Field(i)
			name, opts, _ := strings.Cut(f.Tag.Get("koanf"), ",")
			if !f.IsExported() || name == "-" {
				continue
			}
			ft := f.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if strings.Contains(opts, "squash") {
				walk(ft, prefix)
				continue
			}
			if name == "" {
				name = f.Name
			}
			if prefix != "" {
				name = prefix + "." + name
			}
			switch {
			case kind == 0 && ft.Kind() != reflect.Struct:
				paths[name] = true
				continue
			case ft.Kind() == kind && !(kind == reflect.Slice && ft.Elem().Kind() == reflect.Uint8):
				paths[name] = true
				continue
			}
			walk(ft, name)
		}
	}
	walk(t, "")
	return paths
}

// envKey finds the config key an environment variable names: case and "_"
// do not count, so STEEP_REQUEST__RATE_LIMIT sets request.rateLimit. ok is
// false when it names no key.
func envKey(leaves map[string]bool, name string) (key string, ok bool) {
	norm := func(s string) string { return strings.ToLower(strings.ReplaceAll(s, "_", "")) }
	want := norm(name)
	for p := range leaves {
		if norm(p) == want {
			return p, true
		}
	}
	return strings.ToLower(name), false
}

func unmarshalStrict(k *koanf.Koanf, cfg any) error {
	var md mapstructure.Metadata
	err := k.UnmarshalWithConf("", cfg, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.TextUnmarshallerHookFunc(),
			),
			WeaklyTypedInput: true,
			Metadata:         &md,
			Result:           cfg,
		},
	})
	if err != nil {
		if strings.HasPrefix(err.Error(), "decoding failed") {
			err = errors.Unwrap(err)
		}
		return decodeError(err)
	}
	if len(md.Unused) > 0 {
		return fmt.Errorf("unknown key(s): %s", strings.Join(md.Unused, ", "))
	}
	return nil
}

func decodeError(err error) error {
	if j, ok := err.(interface{ Unwrap() []error }); ok {
		var errs []error
		for _, e := range j.Unwrap() {
			errs = append(errs, decodeError(e))
		}
		return errors.Join(errs...)
	}
	if de, ok := err.(*mapstructure.DecodeError); ok {
		return fmt.Errorf("%s: %w", de.Name(), decodeError(de.Unwrap()))
	}
	return err
}

func Dir(name string) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return name
	}
	return filepath.Join(dir, name)
}

func ExpandHome(path string) string {
	rest, ok := strings.CutPrefix(path, "~/")
	if !ok {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, rest)
}

func expandAll(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != "" {
			out = append(out, ExpandHome(p))
		}
	}
	return out
}

func envPrefix(name string) string {
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	return b.String() + "_"
}

func usage(fs *flag.FlagSet, opts Options) {
	p := opts.EnvPrefix
	if opts.Usage != "" {
		fmt.Fprintln(opts.Stdout, strings.TrimRight(opts.Usage, "\n"))
		fmt.Fprintln(opts.Stdout)
	}
	fmt.Fprintf(opts.Stdout, `Options:
%s
Config files, later wins:
  %s

Every key can also be set from the environment (upper case, "." becomes "__",
words split by "_", comma-separated lists): %sSECTION__SOME_KEY=value
`, fs.FlagUsages(), strings.Join(opts.Files, "\n  "), p)
}
