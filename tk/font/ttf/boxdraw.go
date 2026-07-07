package ttf

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	tk_render "github.com/X11Libre/go-x11proto/tk/render"
)

// Box-drawing glyphs in real TrueType fonts are metrics-correct but often
// small and off-center within their advance cell (a font's box-drawing
// glyphs are drawn like any other glyph, not aligned to a terminal's exact
// cell grid) — see the DejaVu Sans Mono finding in demo/fonttest's history.
// Terminals like kitty, alacritty and foot work around this by hand-drawing
// the box-drawing block against the actual cell rectangle instead of trusting
// the font, which is what DrawString does for the runes covered here.
//
// Only the light and double straight/corner/tee/cross set is covered (21
// runes: U+2500,2502,250C,2510,2514,2518,251C,2524,252C,2534,253C and their
// U+255x double counterparts) — the same set tk/term's DEC Special Graphics
// fallback already special-cases. Heavy lines, dashes, arcs, diagonals,
// block elements (U+2580-259F) and powerline glyphs are NOT covered and fall
// through to the rasterized font glyph.

type lineStyle int

const (
	lineNone lineStyle = iota
	lineSingle
	lineDouble
)

type boxArms struct{ up, down, left, right lineStyle }

var boxDrawTable = map[rune]boxArms{
	'─': {lineNone, lineNone, lineSingle, lineSingle},
	'│': {lineSingle, lineSingle, lineNone, lineNone},
	'┌': {lineNone, lineSingle, lineNone, lineSingle},
	'┐': {lineNone, lineSingle, lineSingle, lineNone},
	'└': {lineSingle, lineNone, lineNone, lineSingle},
	'┘': {lineSingle, lineNone, lineSingle, lineNone},
	'├': {lineSingle, lineSingle, lineNone, lineSingle},
	'┤': {lineSingle, lineSingle, lineSingle, lineNone},
	'┬': {lineNone, lineSingle, lineSingle, lineSingle},
	'┴': {lineSingle, lineNone, lineSingle, lineSingle},
	'┼': {lineSingle, lineSingle, lineSingle, lineSingle},
	'═': {lineNone, lineNone, lineDouble, lineDouble},
	'║': {lineDouble, lineDouble, lineNone, lineNone},
	'╔': {lineNone, lineDouble, lineNone, lineDouble},
	'╗': {lineNone, lineDouble, lineDouble, lineNone},
	'╚': {lineDouble, lineNone, lineNone, lineDouble},
	'╝': {lineDouble, lineNone, lineDouble, lineNone},
	'╠': {lineDouble, lineDouble, lineNone, lineDouble},
	'╣': {lineDouble, lineDouble, lineDouble, lineNone},
	'╦': {lineNone, lineDouble, lineDouble, lineDouble},
	'╩': {lineDouble, lineNone, lineDouble, lineDouble},
	'╬': {lineDouble, lineDouble, lineDouble, lineDouble},
}

// drawBoxChar draws arms directly into the cell rectangle
// [x, x+cellW) x [y, y+cellH) using solid fills — box-drawing lines are
// hard-edged, so no glyph rasterization is needed at all here.
func drawBoxChar(dstPic *tk_render.Picture, x, y, cellW, cellH int, arms boxArms, fg [3]byte) error {
	color := tk_render.Color{
		Red: base.CARD16(fg[0]) * 0x101, Green: base.CARD16(fg[1]) * 0x101, Blue: base.CARD16(fg[2]) * 0x101,
		Alpha: 0xffff,
	}
	cx, cy := x+cellW/2, y+cellH/2
	t := cellH / 12 // single-stroke thickness
	if t < 1 {
		t = 1
	}
	sep := 2 * t // center-to-center separation between a double line's two strokes

	var rects []base.Rectangle
	addRect := func(rx, ry, rw, rh int) {
		if rw > 0 && rh > 0 {
			rects = append(rects, base.Rectangle{X: base.INT16(rx), Y: base.INT16(ry), Width: base.CARD16(rw), Height: base.CARD16(rh)})
		}
	}
	// A vertical stroke of thickness t centered on vx, from vy0 to vy1.
	vstroke := func(vx, vy0, vy1 int) { addRect(vx-t/2, vy0, t, vy1-vy0) }
	// A horizontal stroke of thickness t centered on hy, from hx0 to hx1.
	hstroke := func(hy, hx0, hx1 int) { addRect(hx0, hy-t/2, hx1-hx0, t) }

	// Each arm is drawn from the cell edge to the center (inclusive), so
	// connected arms naturally overlap at the corner/junction instead of
	// needing separate miter geometry. A double arm is two parallel strokes
	// straddling the center line, sep apart.
	switch arms.up {
	case lineSingle:
		vstroke(cx, y, cy+t/2)
	case lineDouble:
		vstroke(cx-sep/2, y, cy+t/2)
		vstroke(cx+sep/2, y, cy+t/2)
	}
	switch arms.down {
	case lineSingle:
		vstroke(cx, cy-t/2, y+cellH)
	case lineDouble:
		vstroke(cx-sep/2, cy-t/2, y+cellH)
		vstroke(cx+sep/2, cy-t/2, y+cellH)
	}
	switch arms.left {
	case lineSingle:
		hstroke(cy, x, cx+t/2)
	case lineDouble:
		hstroke(cy-sep/2, x, cx+t/2)
		hstroke(cy+sep/2, x, cx+t/2)
	}
	switch arms.right {
	case lineSingle:
		hstroke(cy, cx-t/2, x+cellW)
	case lineDouble:
		hstroke(cy-sep/2, cx-t/2, x+cellW)
		hstroke(cy+sep/2, cx-t/2, x+cellW)
	}

	if len(rects) == 0 {
		return nil
	}
	return dstPic.Fill(tk_render.OpOver, color, rects)
}
