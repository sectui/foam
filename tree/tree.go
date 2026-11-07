// Package tree draws entries named by slash-separated paths as a tree of
// folders, with a fuzzy filter. The app draws its leaves and says what opening
// one does.
package tree

import (
	"path"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam/commands"
	"github.com/sectui/foam/draw"
	"github.com/sectui/foam/fuzzy"
	"github.com/sectui/foam/input"
	"github.com/sectui/foam/list"
	"github.com/sectui/foam/theme"
)

// OpenMsg is sent when a leaf is opened.
type OpenMsg struct{ Name string }

// Parent is the folder holding name, "" at the top.
func Parent(name string) string {
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		return name[:i]
	}
	return ""
}

type row[E any] struct {
	e     E
	name  string
	dir   bool
	depth int
}

// Model is a tree of entries E, whose name and kind key tells. Entries come
// sorted as a tree: a folder before what it holds.
type Model[E any] struct {
	// Leaf draws a leaf after its indent; current is the leaf SetCurrent named.
	Leaf func(e E, current bool) string
	// Folder draws a folder's label after its icon; nil draws its last segment.
	Folder func(e E) string
	// Dot is the color of a mark drawn on the left of a leaf; when one has a
	// color, every row gets room for it.
	Dot func(e E) (theme.Color, bool)
	// Empty and NoMatch are shown without entries, and when the filter keeps
	// none.
	Empty, NoMatch string

	key      func(E) (name string, dir bool)
	entries  []E
	expanded map[string]bool
	rows     []row[E]
	list     list.Model

	filter    input.Model
	filtering bool
	current   string
	dotted    bool
}

func New[E any](key func(E) (name string, dir bool)) *Model[E] {
	m := &Model[E]{key: key, expanded: map[string]bool{}, filter: input.New("/ "), Empty: "nothing yet", NoMatch: "nothing matches"}
	m.filter.Placeholder = "filter"
	return m
}

// SetPlaceholder is what the filter shows while empty.
func (m *Model[E]) SetPlaceholder(s string) { m.filter.Placeholder = s }

// SetEntries replaces the tree, keeping the selection on the same entry.
func (m *Model[E]) SetEntries(es []E) {
	sel := m.SelectedName()
	m.entries = es
	m.dotted = false
	if m.Dot != nil {
		for _, e := range es {
			if _, ok := m.Dot(e); ok {
				if _, dir := m.key(e); !dir {
					m.dotted = true
				}
			}
		}
	}
	m.rebuild(sel)
}

// SetCurrent marks the entry name, and reveals it.
func (m *Model[E]) SetCurrent(name string) {
	m.current = name
	m.Reveal(name)
}

// Reveal opens the folders above name and selects it.
func (m *Model[E]) Reveal(name string) {
	for d := Parent(name); d != ""; d = Parent(d) {
		m.expanded[d] = true
	}
	m.rebuild(name)
}

func (m *Model[E]) Selected() (e E, ok bool) {
	if len(m.rows) == 0 {
		return e, false
	}
	return m.rows[m.list.Selected()].e, true
}

// SelectedName is the name of the selected entry, "" without one.
func (m *Model[E]) SelectedName() string {
	if len(m.rows) == 0 {
		return ""
	}
	return m.rows[m.list.Selected()].name
}

// SelectedDir tells whether the selection is a folder.
func (m *Model[E]) SelectedDir() bool {
	return len(m.rows) > 0 && m.rows[m.list.Selected()].dir
}

// Dir is the folder new entries go in: the selected folder, or the folder of
// the selected leaf.
func (m *Model[E]) Dir() string {
	if m.SelectedDir() {
		return m.SelectedName()
	}
	return Parent(m.SelectedName())
}

func (m *Model[E]) rebuild(keep string) {
	m.rows = m.rows[:0]
	q := m.filter.Value()
	var keepDirs map[string]bool
	if q != "" {
		keepDirs = map[string]bool{}
		for _, e := range m.entries {
			name, dir := m.key(e)
			if _, _, ok := fuzzy.Match(q, name); ok && !dir {
				for d := Parent(name); d != ""; d = Parent(d) {
					keepDirs[d] = true
				}
				keepDirs[name] = true
			}
		}
	}
	for _, e := range m.entries {
		name, dir := m.key(e)
		if (q == "" && m.visible(name)) || keepDirs[name] {
			m.rows = append(m.rows, row[E]{e, name, dir, strings.Count(name, "/")})
		}
	}
	m.list.SetLen(len(m.rows))
	for i, r := range m.rows {
		if r.name == keep {
			m.list.Select(i)
			break
		}
	}
}

func (m *Model[E]) visible(name string) bool {
	for d := Parent(name); d != ""; d = Parent(d) {
		if !m.expanded[d] {
			return false
		}
	}
	return true
}

// Capturing tells whether the filter is being typed: it takes the keys.
func (m *Model[E]) Capturing() bool { return m.filtering }

// Input handles a key while the filter is being typed.
func (m *Model[E]) Input(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "enter", "esc", "ctrl+c":
		m.filtering = false
	case "up", "ctrl+k", "ctrl+p":
		m.list.Move(-1)
	case "down", "ctrl+j", "ctrl+n":
		m.list.Move(1)
	default:
		if !m.filter.Update(msg) {
			return false
		}
		m.refilter()
	}
	return true
}

func (m *Model[E]) Paste(s string) {
	if m.filtering {
		m.filter.Update(tea.PasteMsg{Content: s})
		m.refilter()
	}
}

// refilter applies the filter typed, keeping the selected leaf when it still
// matches, the first leaf otherwise.
func (m *Model[E]) refilter() {
	keep := ""
	if !m.SelectedDir() {
		keep = m.SelectedName()
	}
	m.rebuild(keep)
	if m.SelectedName() != keep || keep == "" {
		m.list.Select(m.firstLeaf())
	}
}

func (m *Model[E]) firstLeaf() int {
	for i, r := range m.rows {
		if !r.dir {
			return i
		}
	}
	return 0
}

func (m *Model[E]) setExpanded(open bool) tea.Cmd {
	if len(m.rows) == 0 {
		return nil
	}
	name := m.SelectedName()
	if !m.SelectedDir() {
		if open {
			return nil
		}
		// h on a leaf goes to its folder
		if d := Parent(name); d != "" {
			m.expanded[d] = false
			m.rebuild(d)
		}
		return nil
	}
	m.expanded[name] = open
	m.rebuild(name)
	return nil
}

func (m *Model[E]) setAll(open bool) tea.Cmd {
	for _, e := range m.entries {
		if name, dir := m.key(e); dir {
			m.expanded[name] = open
		}
	}
	keep := m.SelectedName()
	if !open {
		for keep != "" && !m.visible(keep) {
			keep = Parent(keep)
		}
	}
	m.rebuild(keep)
	return nil
}

func (m *Model[E]) open() tea.Cmd {
	if len(m.rows) == 0 {
		return nil
	}
	name := m.SelectedName()
	if m.SelectedDir() {
		m.expanded[name] = !m.expanded[name]
		m.rebuild(name)
		return nil
	}
	return func() tea.Msg { return OpenMsg{name} }
}

// Commands are bound under tree.*, and list.* for the moves.
func (m *Model[E]) Commands() []commands.Command {
	return append(m.list.Commands(),
		commands.Command{ID: "tree.open", Help: "open", Run: m.open},
		commands.Command{ID: "tree.expand", Run: func() tea.Cmd { return m.setExpanded(true) }},
		commands.Command{ID: "tree.collapse", Run: func() tea.Cmd { return m.setExpanded(false) }},
		commands.Command{ID: "tree.expand-all", Title: "Expand every folder", Run: func() tea.Cmd { return m.setAll(true) }},
		commands.Command{ID: "tree.collapse-all", Title: "Collapse every folder", Run: func() tea.Cmd { return m.setAll(false) }},
		commands.Command{ID: "tree.filter", Help: "filter", Run: func() tea.Cmd { m.filtering = true; return nil }},
		commands.Command{ID: "tree.clear-filter", When: func() bool { return m.filter.Value() != "" },
			Run: func() tea.Cmd {
				sel := m.SelectedName()
				m.filter.Reset()
				m.rebuild(sel)
				return nil
			}},
	)
}

// Mouse handles an event in the pane. A click while the pane is not focused
// only moves the cursor there; otherwise it opens a leaf, or a folder clicked
// twice.
func (m *Model[E]) Mouse(ev tea.MouseMsg, focused bool) tea.Cmd {
	top := 0
	if m.showFilter() {
		top = 2
	}
	mm := ev.Mouse()
	mm.Y -= top
	if _, ok := ev.(tea.MouseClickMsg); ok {
		before := m.list.Selected()
		if i := m.list.Mouse(tea.MouseClickMsg(mm)); i >= 0 && focused && (i == before || !m.rows[i].dir) {
			return m.open()
		}
		return nil
	}
	if w, ok := ev.(tea.MouseWheelMsg); ok {
		m.list.Mouse(w)
	}
	return nil
}

func (m *Model[E]) showFilter() bool { return m.filtering || m.filter.Value() != "" }

func (m *Model[E]) Scroll() draw.Scroll { return m.list.Scroll() }

func (m *Model[E]) View(w, h int, focused bool) string {
	var lines []string
	if m.showFilter() {
		f := m.filter
		if !m.filtering {
			f.Update(tea.KeyPressMsg{Code: tea.KeyHome})
		}
		lines = append(lines, f.View(w, m.filtering), draw.Rule(w))
		h -= 2
	}
	m.list.SetHeight(max(h, 0))
	if len(m.rows) == 0 {
		msg := m.Empty
		if m.filter.Value() != "" {
			msg = m.NoMatch
		}
		return strings.Join(append(lines, theme.Subtle().Render(" "+ansi.Truncate(msg, w-1, "…"))), "\n")
	}
	lead := " "
	if m.dotted {
		lead = "   "
	}
	start, end := m.list.Visible()
	for i := start; i < end; i++ {
		r := m.rows[i]
		indent := strings.Repeat("  ", r.depth)
		var s string
		if r.dir {
			icon := theme.Icon("\uf07b ", "▸ ")
			if m.expanded[r.name] || m.filter.Value() != "" {
				icon = theme.Icon("\uf07c ", "▾ ")
			}
			label := theme.Accent().Render(draw.Clean(path.Base(r.name)))
			if m.Folder != nil {
				label = m.Folder(r.e)
			}
			s = indent + theme.Accent().Render(icon) + label
		} else if m.Leaf != nil {
			s = indent + m.Leaf(r.e, r.name == m.current)
		} else {
			s = indent + theme.Fg().Render(draw.Clean(path.Base(r.name)))
		}
		s = ansi.Truncate(lead+s, w, "…")
		sel := i == m.list.Selected()
		if sel {
			s = draw.Selected(s, w, focused)
		}
		if m.dotted && !r.dir && w >= 3 {
			if c, ok := m.Dot(r.e); ok {
				// drawn over the selection, which takes the colors away
				st := c.Style()
				if sel {
					bg := theme.Selection()
					if !focused {
						bg = theme.Surface()
					}
					st = st.Background(bg.Value())
				}
				s = ansi.Cut(s, 0, 1) + st.Render(theme.Icon("\uf111", "●")) + ansi.Cut(s, 2, w)
			}
		}
		lines = append(lines, s)
	}
	return strings.Join(lines, "\n")
}
