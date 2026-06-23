// Package font wraps a core X11 server font together with the metrics needed to
// lay out text: glyph advances, ascent and descent. A *Font both measures and
// draws strings, so it satisfies the TextRenderer interface used by tk/widget
// and serves as the text backend for the editor widgets.
package font

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

// Font is a server font plus its cached metrics.
type Font struct {
	ID      base.FONT
	Ascent  int
	Descent int

	// Glyph advances. For a fixed-width font widths is nil and fixedW holds the
	// single advance; otherwise widths[char-firstChar] is the advance and
	// defaultW covers characters outside the range.
	firstChar int
	widths    []int
	fixedW    int
	defaultW  int
}

// Open opens the named core font (an XLFD or alias like "fixed") and queries its
// metrics. The caller owns the font and should Close it.
func Open(conn *core.X11Conn, name string) (*Font, error) {
	id, err := rpc.OpenFont(conn, name)
	if err != nil {
		return nil, err
	}
	f, err := Query(conn, id)
	if err != nil {
		_ = rpc.CloseFont(conn, id)
		return nil, err
	}
	return f, nil
}

// Query builds a Font from an already-opened font id (e.g. one returned by
// theme.OpenFont) by issuing QueryFont for its metrics.
func Query(conn *core.X11Conn, id base.FONT) (*Font, error) {
	rep, err := rpc.QueryFont(conn, id)
	if err != nil {
		return nil, err
	}
	f := &Font{
		ID:       id,
		Ascent:   int(rep.FontAscent),
		Descent:  int(rep.FontDescent),
		defaultW: int(rep.MaxBounds.CharacterWidth),
	}
	if len(rep.CharInfos) == 0 {
		// No per-char metrics: a fixed-width font.
		f.fixedW = int(rep.MaxBounds.CharacterWidth)
	} else {
		f.firstChar = int(rep.MinCharOrByte2)
		f.widths = make([]int, len(rep.CharInfos))
		for i, ci := range rep.CharInfos {
			f.widths[i] = int(ci.CharacterWidth)
		}
	}
	return f, nil
}

// Close releases the server font.
func (f *Font) Close(conn *core.X11Conn) error { return rpc.CloseFont(conn, f.ID) }

// Height is the line height (ascent + descent).
func (f *Font) Height() int { return f.Ascent + f.Descent }

// RuneWidth returns the advance of r. Core fonts are byte-indexed (Latin-1), so
// runes above 0xff fall back to the font's default width.
func (f *Font) RuneWidth(r rune) int {
	if f.fixedW > 0 {
		return f.fixedW
	}
	idx := int(r) - f.firstChar
	if idx >= 0 && idx < len(f.widths) {
		if w := f.widths[idx]; w > 0 {
			return w
		}
	}
	return f.defaultW
}

// TextWidth returns the pixel width of s.
func (f *Font) TextWidth(s string) int {
	w := 0
	for _, r := range s {
		w += f.RuneWidth(r)
	}
	return w
}

// IndexAtX returns the character index in s whose left edge is closest to pixel
// offset x (used to place a caret from a mouse click). The result is in [0,
// len([]rune(s))].
func (f *Font) IndexAtX(s string, x int) int {
	if x <= 0 {
		return 0
	}
	acc, i := 0, 0
	for _, r := range s {
		w := f.RuneWidth(r)
		if x < acc+w/2+1 {
			return i
		}
		acc += w
		i++
	}
	return i
}

// DrawText draws s with its top-left corner at (x, y), so callers position by
// the cell box rather than the baseline. It draws foreground only (PutText8),
// leaving the background untouched; the GC's font must be this font (see
// SetOn). scale is ignored — core fonts have a fixed size. Satisfies
// tk/widget.TextRenderer.
func (f *Font) DrawText(d tk_core.Drawable, gc base.GC, x, y base.INT16, scale int, s string) error {
	return d.PutText8(gc, x, y+base.INT16(f.Ascent), s)
}

// DrawTextBG draws s like DrawText but also fills the glyph cells with the GC's
// background colour (ImageText8), which cleanly overwrites previous text — handy
// for redrawing an editor line in place.
func (f *Font) DrawTextBG(d tk_core.Drawable, gc base.GC, x, y base.INT16, s string) error {
	return d.ImageText8(gc, x, y+base.INT16(f.Ascent), s)
}

// Measure returns the pixel size s occupies. Satisfies tk/widget.TextRenderer.
func (f *Font) Measure(scale int, s string) (w, h int) {
	return f.TextWidth(s), f.Height()
}

// SetOn sets this font on a GC so its text-drawing requests use it.
func (f *Font) SetOn(gc *tk_core.GC) error { return gc.SetFont(f.ID) }
