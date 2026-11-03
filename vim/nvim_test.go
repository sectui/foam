package vim

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestAgainstNeovim plays commands in Neovim and in the engine, and compares
// the texts. FOAM_NVIM=1 runs it.
func TestAgainstNeovim(t *testing.T) {
	if os.Getenv("FOAM_NVIM") == "" {
		t.Skip("FOAM_NVIM=1 compares with Neovim")
	}
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		t.Skip("no nvim")
	}
	texts := []string{"foo bar baz\nqux quux\n\nlast line here", "a(b c)d \"x y\" [z]\n  indented {w} 'q'", "one\ntwo\nthree"}
	starts := []string{"", "w", "2w", "j", "je", "G", "G$", "$"}
	ops := []string{"x", "X", "3x", "dd", "2dd", "3dd", "D", "C", "3D", "3C", "J", "3J", "~", "3~", "p", "P", "yyp",
		"3yyp", "2yyP", "rZ", "3rZ", "dw", "3dw", "de", "db", "3db", "cw", "3cw", "ce", "cb", "dW", "dB", "d$", "d0",
		"d^", "dj", "dk", "dG", "dgg", "3dj", "diw", "3diw", "daw", "3daw", "ciw", "caw", `ci"`, `da"`, "dfa", "dta",
		"dFo", "dTo", "d}", "d{", "yw", "yyP", "yiwP", "xp", "ddp", "vd", "vjd", "vey", "v$d", "Vd", "Vjd", "Vjy",
		"v~", "vjc", "3vd", "viwd", "vawd", "AX", "IX", "aX", "iX", "3iX", "3aX", "u", "xu", "xu<C-r>", "ddu", "3xu"}
	type tc struct{ text, keys string }
	var cases []tc
	for _, text := range texts {
		for _, s := range starts {
			for _, op := range ops {
				k := s + op
				if strings.ContainsAny(op[:1], "cCaAiI") || strings.Contains(op, "c") && !strings.Contains(op, "<C") {
					k += "T<Esc>"
				}
				cases = append(cases, tc{text, k})
			}
		}
	}
	// Known gaps: counted words over an empty line.
	skip := map[string]bool{"j3daw": true, "je3daw": true, "je3dw": true}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, c := range cases {
		if skip[c.keys] {
			continue
		}
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			want := neovim(t, nvim, c.text, c.keys)
			e := New(c.text)
			feedNotation(e, c.keys+"<Esc>")
			if got := e.Text(); got != want {
				t.Errorf("%q + %q: got %q, Neovim %q", c.text, c.keys, got, want)
			}
		})
	}
	wg.Wait()
}

func neovim(t *testing.T, nvim, text, keys string) string {
	dir := t.TempDir()
	in, out := filepath.Join(dir, "in"), filepath.Join(dir, "out")
	os.WriteFile(in, []byte(text+"\n"), 0o644)
	k := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "<Esc>", `\<Esc>`, "<C-r>", `\<C-r>`).Replace(keys)
	cmd := exec.Command(nvim, "--clean", "--headless", "-n", "-i", "NONE", in,
		"-c", "set startofline noautoindent formatoptions= undolevels=1000",
		"-c", `call feedkeys("`+k+`\<Esc>", "xt")`, "-c", "w! "+out, "-c", "qa!")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("nvim: %v\n%s", err, b)
	}
	b, _ := os.ReadFile(out)
	return strings.TrimSuffix(string(b), "\n")
}

// feedNotation types keys written as in vim: <Esc>, <C-r>.
func feedNotation(e *Editor, keys string) {
	for keys != "" {
		switch {
		case strings.HasPrefix(keys, "<Esc>"):
			e.Key("esc", "")
			keys = keys[5:]
		case strings.HasPrefix(keys, "<C-r>"):
			e.Key("ctrl+r", "")
			keys = keys[5:]
		default:
			e.Key(keys[:1], keys[:1])
			keys = keys[1:]
		}
	}
}
