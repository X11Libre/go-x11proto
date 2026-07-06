package widget

import (
	"sort"
	"strings"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/tk/font"
)

// Span describes one styled, optionally clickable run of runes on a line, as
// returned by a Highlighter. Start/End are rune offsets [Start, End) into the
// line. A zero Font/Fg means "use the TextView's own Font/Fg" — a highlighter
// only needs to return spans for the runs it wants to style differently, not
// the whole line.
type Span struct {
	Start, End int
	Font       *font.Font  // nil = TextView.Font (e.g. a bold/italic variant)
	Fg         base.CARD32 // 0 = TextView.Fg
	Link       string      // non-empty marks this run clickable; see TextView.OnLink
}

// Highlighter splits one line into styled spans. It is called once per visible
// line on every Draw, so it should be cheap — a real syntax highlighter
// belongs in its own package and should cache/incrementally re-lex by lineIdx
// rather than re-tokenizing the whole buffer every frame. Gaps between (and
// around) the returned spans are drawn in the TextView's own Font/Fg, so a
// highlighter only needs to report the runs it wants to style.
//
// Highlighter is deliberately just a function value, not an interface with a
// registry: a program that never sets TextView.Highlighter never imports
// whatever syntax/markdown package would supply one, so it pays nothing —
// neither code size nor runtime cost — for a feature it doesn't use.
type Highlighter func(lineIdx int, line string) []Span

// normalizeSpans sorts spans by Start, clips them to [0, n), drops empty/
// invalid ones, clips overlaps to the earlier span's End, and fills every gap
// with a zero-value Span (default Font/Fg) so the result is an ordered,
// gapless run list covering the whole line — exactly what Draw's loop needs,
// regardless of how sloppy or partial the Highlighter's own output is.
func normalizeSpans(spans []Span, n int) []Span {
	clean := make([]Span, 0, len(spans))
	for _, s := range spans {
		if s.Start < 0 {
			s.Start = 0
		}
		if s.End > n {
			s.End = n
		}
		if s.Start < s.End {
			clean = append(clean, s)
		}
	}
	sort.Slice(clean, func(i, j int) bool { return clean[i].Start < clean[j].Start })

	out := make([]Span, 0, len(clean)*2+1)
	pos := 0
	for _, s := range clean {
		if s.Start > pos {
			out = append(out, Span{Start: pos, End: s.Start})
		}
		if s.Start < pos {
			s.Start = pos // overlaps an earlier span: clip instead of drawing twice
		}
		if s.Start < s.End {
			out = append(out, s)
			pos = s.End
		}
	}
	if pos < n {
		out = append(out, Span{Start: pos, End: n})
	}
	return out
}

// spanLinkAt returns the Link of the span covering rune column col, if any.
// Pulled out of LinkAt so the hit-testing logic is unit-testable without a
// font/pixel round trip.
func spanLinkAt(spans []Span, col int) (string, bool) {
	for _, s := range spans {
		if s.Link != "" && col >= s.Start && col < s.End {
			return s.Link, true
		}
	}
	return "", false
}

// expandRange returns line's runes [start, end) with tabs expanded to spaces
// at their true column in the full line — unlike expanding the substring on
// its own, a tab that falls inside the range still lands on the same stop it
// would in the whole, unsplit line.
func (t *TextView) expandRange(line string, start, end int) string {
	tw := t.tabCols()
	var b strings.Builder
	col := 0
	for i, r := range []rune(line) {
		if i >= end {
			break
		}
		if r == '\t' {
			n := tw - col%tw
			if i >= start {
				b.WriteString(strings.Repeat(" ", n))
			}
			col += n
		} else {
			if i >= start {
				b.WriteRune(r)
			}
			col++
		}
	}
	return b.String()
}
