package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	flag "github.com/spf13/pflag"
)

type look struct {
	Border string `koanf:"borderStyle" validate:"oneof=rounded heavy"`
}

type testCfg struct {
	look  `koanf:",squash"`
	Name  string         `koanf:"name"`
	Rate  int            `koanf:"rateLimit"`
	Tags  []string       `koanf:"tags"`
	Keys  map[string]any `koanf:"keys"`
	Where string         `koanf:"where"`
}

var testDefaults = []byte("borderStyle: rounded\nname: a\nrateLimit: 0\ntags: []\nkeys: {}\nwhere: ''\n")

func load(t *testing.T, opts Options, args ...string) (*testCfg, error) {
	t.Helper()
	opts.Name = "foamtest"
	opts.Stdout = io.Discard
	opts.Base = append([][]byte{testDefaults}, opts.Base...)
	cfg, _, err := Load[testCfg](opts, args)
	return cfg, err
}

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSetUnderMap(t *testing.T) {
	cfg, err := load(t, Options{}, "-c", "keys.collection.delete=x,delete")
	if err != nil {
		t.Fatal(err)
	}
	col, _ := cfg.Keys["collection"].(map[string]any)
	if col["delete"] != "x,delete" {
		t.Fatalf("keys = %#v", cfg.Keys)
	}
}

func TestSetUnknownRejected(t *testing.T) {
	if _, err := load(t, Options{}, "--set", "nope=1"); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("got %v", err)
	}
}

func TestLayers(t *testing.T) {
	dir := t.TempDir()
	low := write(t, dir, "low.yaml", "name: low\ntags: [x]\n")
	high := write(t, dir, "high.yaml", "name: high\n")
	project := filepath.Join(dir, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, project, "config.yaml", "rateLimit: 7\nborderStyle: heavy\n")
	t.Setenv("FOAMTEST_WHERE", project)

	cfg, err := load(t, Options{
		Files: []string{low, filepath.Join(dir, "missing.yaml"), high},
		Extra: func(get func(string) string) []string {
			return []string{filepath.Join(get("where"), "config.yaml")}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "high" || len(cfg.Tags) != 1 || cfg.Rate != 7 || cfg.Border != "heavy" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestFlagKeysAndEnv(t *testing.T) {
	t.Setenv("FOAMTEST_RATE_LIMIT", "3")
	t.Setenv("FOAMTEST_TAGS", "a, b")
	var rate int
	opts := Options{
		Flags:    func(fs *flag.FlagSet) { fs.IntVar(&rate, "rate", 0, "") },
		FlagKeys: map[string]string{"rate": "rateLimit"},
	}
	cfg, err := load(t, opts)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Rate != 3 || strings.Join(cfg.Tags, "|") != "a|b" {
		t.Fatalf("env: cfg = %+v", cfg)
	}
	if cfg, err = load(t, opts, "--rate", "9"); err != nil || cfg.Rate != 9 {
		t.Fatalf("flag: cfg = %+v, err = %v", cfg, err)
	}
}

func TestUnknownKeyInFile(t *testing.T) {
	f := write(t, t.TempDir(), "c.yaml", "nmae: typo\n")
	if _, err := load(t, Options{Files: []string{f}}); err == nil || !strings.Contains(err.Error(), "nmae") {
		t.Fatalf("got %v", err)
	}
}

func TestDottedKeys(t *testing.T) {
	f := write(t, t.TempDir(), "c.yaml", "keys:\n  app.quit: x\n")
	cfg, err := load(t, Options{Base: [][]byte{[]byte("keys: {app: {quit: q}}\n")}, Files: []string{f}})
	if err != nil {
		t.Fatal(err)
	}
	app, _ := cfg.Keys["app"].(map[string]any)
	if len(cfg.Keys) != 1 || app["quit"] != "x" {
		t.Fatalf("keys = %#v", cfg.Keys)
	}
}

func TestErrorPathUnderSquash(t *testing.T) {
	_, err := load(t, Options{}, "-c", "borderStyle=thin")
	if err == nil || !strings.Contains(err.Error(), "\nborderStyle:") {
		t.Fatalf("got %v", err)
	}
}
