package shot

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

type face struct {
	f    *sfnt.Font
	face font.Face
	buf  sfnt.Buffer
}

func (f *face) has(r rune) bool {
	i, err := f.f.GlyphIndex(&f.buf, r)
	return err == nil && i != 0
}

type style int

const (
	regular style = iota
	bold
	italic
	boldItalic
)

var stylePatterns = [4]string{"", ":bold", ":italic", ":bold:italic"}

// fonts are the faces of a family at one size, and the fonts fontconfig
// falls back on for what the family lacks.
type fonts struct {
	px       float64
	family   string // fontconfig's pattern, "" for a font file
	name     string // the family's name, for SVG
	styles   [4]*face
	fakeBold [4]bool // the face drawn is not bold: thicken it
	byFile   map[string]*face
	bad      map[string]error // files that would not parse
	byRune   map[rune]*face
}

func loadFonts(spec string, px float64) (*fonts, error) {
	fs := &fonts{px: px, byFile: map[string]*face{}, bad: map[string]error{}, byRune: map[rune]*face{}}
	if isFile(spec) {
		f, err := fs.open(spec, 0)
		if err != nil {
			return nil, err
		}
		fs.name, _ = f.f.Name(&f.buf, sfnt.NameIDFamily)
		for s := range fs.styles {
			fs.styles[s] = f
			fs.fakeBold[s] = s == int(bold) || s == int(boldItalic)
		}
		return fs, nil
	}
	fs.family = spec
	if fs.family == "" {
		fs.family = "monospace"
	}
	var files [4]string
	for s, p := range stylePatterns {
		out, err := fcMatch(fs.family+p, "%{file}\t%{index}\t%{family}")
		if err != nil {
			return nil, err
		}
		parts := strings.SplitN(out, "\t", 3)
		if len(parts) < 3 {
			return nil, fmt.Errorf("fc-match %q: unexpected output %q", fs.family, out)
		}
		idx, _ := strconv.Atoi(parts[1])
		f, err := fs.open(parts[0], idx)
		if err != nil {
			return nil, err
		}
		if s == int(regular) {
			fs.name, _, _ = strings.Cut(parts[2], ",")
		}
		files[s] = parts[0] + "\t" + parts[1]
		fs.styles[s] = f
	}
	// a family without bold gets its regular back
	fs.fakeBold[bold] = files[bold] == files[regular]
	fs.fakeBold[boldItalic] = files[boldItalic] == files[italic]
	return fs, nil
}

func isFile(spec string) bool {
	if spec == "" {
		return false
	}
	switch strings.ToLower(filepath.Ext(spec)) {
	case ".ttf", ".otf", ".ttc", ".otc":
		return true
	}
	return strings.ContainsRune(spec, os.PathSeparator)
}

func fcMatch(pattern, format string, flags ...string) (string, error) {
	out, err := exec.Command("fc-match", append(flags, "-f", format, pattern)...).Output()
	if errors.Is(err, exec.ErrNotFound) {
		return "", errors.New("fc-match (fontconfig) not found: set the font to a font file")
	}
	if err != nil {
		return "", fmt.Errorf("fc-match %q: %w", pattern, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (fs *fonts) open(path string, index int) (*face, error) {
	index &= 0xffff // the upper bits pick a variable font's instance
	key := path + "\t" + strconv.Itoa(index)
	if f, ok := fs.byFile[key]; ok {
		return f, nil
	}
	if err := fs.bad[key]; err != nil {
		return nil, err
	}
	f, err := fs.parse(path, index)
	if err != nil {
		fs.bad[key] = err
		return nil, err
	}
	fs.byFile[key] = f
	return f, nil
}

func (fs *fonts) parse(path string, index int) (*face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("font: %w", err)
	}
	var sf *sfnt.Font
	if c, err := opentype.ParseCollection(data); err == nil {
		sf, err = c.Font(index)
		if err != nil {
			return nil, fmt.Errorf("font %s: %w", path, err)
		}
	} else if sf, err = opentype.Parse(data); err != nil {
		return nil, fmt.Errorf("font %s: %w", path, err)
	}
	ff, err := opentype.NewFace(sf, &opentype.FaceOptions{Size: fs.px, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, fmt.Errorf("font %s: %w", path, err)
	}
	return &face{f: sf, face: ff}, nil
}

// pick returns the face drawing r in a style, and whether to thicken it.
func (fs *fonts) pick(r rune, s style) (*face, bool) {
	if f := fs.styles[s]; f.has(r) {
		return f, fs.fakeBold[s]
	}
	if f := fs.styles[regular]; f.has(r) {
		return f, s == bold || s == boldItalic
	}
	if fs.family == "" {
		return nil, false
	}
	f, ok := fs.byRune[r]
	if !ok {
		f = fs.fallback(r)
		fs.byRune[r] = f
	}
	return f, false
}

// fallback asks fontconfig for a font holding r, monospaced ones first.
func (fs *fonts) fallback(r rune) *face {
	out, err := exec.Command("fc-list", "-f", "%{spacing}\t%{file}\t%{index}\n", fmt.Sprintf(":charset=%x", r)).Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	slices.SortStableFunc(lines, func(a, b string) int {
		return cmp.Compare(boolInt(!strings.HasPrefix(a, "100\t")), boolInt(!strings.HasPrefix(b, "100\t")))
	})
	for i, line := range lines {
		if i == 8 {
			break
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			continue
		}
		n, _ := strconv.Atoi(parts[2])
		if f, err := fs.open(parts[1], n); err == nil && f.has(r) {
			return f
		}
	}
	return nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// cell is the size of a cell in pixels and the baseline's height in it.
func (fs *fonts) cell() (w, h, baseline int) {
	f := fs.styles[regular].face
	adv, ok := f.GlyphAdvance('M')
	if !ok {
		adv = fixed.I(int(math.Round(fs.px * 0.6)))
	}
	m := f.Metrics()
	w = max(adv.Round(), 1)
	h = max(m.Height.Ceil(), (m.Ascent + m.Descent).Ceil(), 1)
	baseline = m.Ascent.Round() + (h-(m.Ascent+m.Descent).Round())/2
	return w, h, baseline
}
