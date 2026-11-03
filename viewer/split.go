package viewer

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// splitRows cuts a styled line into rows of w cells in one pass. Each row
// ends with a reset and starts again with the styles in force, so that rows
// stand alone.
func splitRows(s string, w int) []string {
	if w <= 0 {
		return []string{s}
	}
	var (
		rows   []string
		row    strings.Builder
		active strings.Builder // SGR sequences in force since the last reset
		width  int
		state  byte
	)
	for len(s) > 0 {
		seq, sw, n, newState := ansi.DecodeSequence(s, state, nil)
		state = newState
		s = s[n:]
		if sw == 0 {
			if strings.HasPrefix(seq, "\x1b[") && strings.HasSuffix(seq, "m") {
				if seq == "\x1b[m" || seq == "\x1b[0m" {
					active.Reset()
				} else {
					active.WriteString(seq)
				}
			}
			row.WriteString(seq)
			continue
		}
		if width+sw > w {
			row.WriteString("\x1b[m")
			rows = append(rows, row.String())
			row.Reset()
			row.WriteString(active.String())
			width = 0
		}
		row.WriteString(seq)
		width += sw
	}
	return append(rows, row.String())
}
