package widget

import (
	"reflect"
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/tk/font"
)

func TestNormalizeSpansFillsGaps(t *testing.T) {
	bold := &font.Font{Ascent: 10, Descent: 2}
	got := normalizeSpans([]Span{{Start: 2, End: 4, Font: bold}}, 6)
	want := []Span{
		{Start: 0, End: 2},
		{Start: 2, End: 4, Font: bold},
		{Start: 4, End: 6},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeSpans = %+v, want %+v", got, want)
	}
}

func TestNormalizeSpansNoSpansCoversWholeLine(t *testing.T) {
	got := normalizeSpans(nil, 5)
	want := []Span{{Start: 0, End: 5}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeSpans(nil) = %+v, want %+v", got, want)
	}
}

func TestNormalizeSpansFullCoverageNoGaps(t *testing.T) {
	red := base.CARD32(0xff0000)
	got := normalizeSpans([]Span{{Start: 0, End: 3, Fg: red}, {Start: 3, End: 5}}, 5)
	want := []Span{{Start: 0, End: 3, Fg: red}, {Start: 3, End: 5}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeSpans = %+v, want %+v", got, want)
	}
}

func TestNormalizeSpansClipsOutOfRange(t *testing.T) {
	got := normalizeSpans([]Span{{Start: -3, End: 2}, {Start: 4, End: 100}}, 6)
	want := []Span{
		{Start: 0, End: 2},
		{Start: 2, End: 4},
		{Start: 4, End: 6},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeSpans = %+v, want %+v", got, want)
	}
}

func TestNormalizeSpansDropsEmptyAndClipsOverlap(t *testing.T) {
	got := normalizeSpans([]Span{{Start: 3, End: 3}, {Start: 0, End: 4}, {Start: 2, End: 6}}, 6)
	want := []Span{
		{Start: 0, End: 4},
		{Start: 4, End: 6}, // second span clipped from 2 to the first span's End
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeSpans = %+v, want %+v", got, want)
	}
}

func TestNormalizeSpansUnsortedInput(t *testing.T) {
	got := normalizeSpans([]Span{{Start: 4, End: 6}, {Start: 0, End: 2}}, 6)
	want := []Span{
		{Start: 0, End: 2},
		{Start: 2, End: 4},
		{Start: 4, End: 6},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeSpans = %+v, want %+v", got, want)
	}
}

func TestSpanLinkAt(t *testing.T) {
	spans := []Span{
		{Start: 0, End: 3},
		{Start: 3, End: 8, Link: "http://example.invalid"},
		{Start: 8, End: 10},
	}
	if _, ok := spanLinkAt(spans, 1); ok {
		t.Errorf("column 1 should not be linked")
	}
	if id, ok := spanLinkAt(spans, 3); !ok || id != "http://example.invalid" {
		t.Errorf("column 3 (span start) = %q,%v, want the link", id, ok)
	}
	if id, ok := spanLinkAt(spans, 7); !ok || id != "http://example.invalid" {
		t.Errorf("column 7 (span interior) = %q,%v, want the link", id, ok)
	}
	if _, ok := spanLinkAt(spans, 8); ok {
		t.Errorf("column 8 (span End, exclusive) should not be linked")
	}
	if _, ok := spanLinkAt(spans, 9); ok {
		t.Errorf("column 9 should not be linked")
	}
}

func TestExpandRangeMatchesExpandFromStart(t *testing.T) {
	tv := newTV(5)
	line := "a\tbc"
	full := tv.expand(line, -1)
	if got := tv.expandRange(line, 0, len([]rune(line))); got != full {
		t.Errorf("expandRange(0,end) = %q, want %q (== expand(-1))", got, full)
	}
}

func TestExpandRangeMidLineTabAlignsToTrueColumn(t *testing.T) {
	tv := newTV(5)
	tv.TabWidth = 4
	line := "ab\tcd" // tab at rune index 2, expands to the 4-column stop -> 2 spaces
	full := tv.expand(line, -1)
	if full != "ab  cd" {
		t.Fatalf("sanity: expand(-1) = %q, want %q", full, "ab  cd")
	}
	// The range [2,5) covers "\tcd"; on its own (col reset to 0) a naive
	// expand would emit 4 spaces instead of 2, since it wouldn't know the tab
	// starts at true column 2.
	if got := tv.expandRange(line, 2, len([]rune(line))); got != "  cd" {
		t.Errorf("expandRange(2,end) = %q, want %q", got, "  cd")
	}
	// The range before the tab is unaffected.
	if got := tv.expandRange(line, 0, 2); got != "ab" {
		t.Errorf("expandRange(0,2) = %q, want %q", got, "ab")
	}
}

func TestLinkAtNoHighlighterReturnsFalse(t *testing.T) {
	tv := newTV(5, "hello")
	if _, ok := tv.LinkAt(0, 0); ok {
		t.Errorf("LinkAt with no Highlighter should never match")
	}
}

func TestLinkAtOutOfRangeRow(t *testing.T) {
	tv := newTV(5, "hello")
	tv.Highlighter = func(lineIdx int, line string) []Span {
		return []Span{{Start: 0, End: len([]rune(line)), Link: "x"}}
	}
	// y below the last line, past VisibleLines()*Height, maps past len(lines).
	if _, ok := tv.LinkAt(0, 100000); ok {
		t.Errorf("LinkAt should reject a row past the end of the buffer")
	}
}
