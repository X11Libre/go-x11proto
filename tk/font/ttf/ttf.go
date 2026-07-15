// Package ttf renders antialiased TrueType/OpenType text on go-x11proto
// drawables. It rasterizes glyphs with golang.org/x/image/font/{sfnt,
// opentype} — a pure-Go rasterizer, no cgo/libfreetype — and composites each
// glyph's alpha mask onto a destination via the RENDER extension (tk/render),
// the same technique Xft uses internally with real FreeType.
//
// This is a from-scratch sibling of tk/font, not a replacement: tk/font
// wraps X core server fonts (bitmap, unantialiased, no RENDER dependency);
// Face here needs RENDER and draws through a *tk_render.Picture instead of a
// GC, so it does not implement tk/widget's TextRenderer interface (which is
// GC-based) — callers that want antialiased text talk to Face directly.
//
// Rasterized glyphs and per-color solid-fill source pictures are cached for
// the Face's lifetime (Close frees everything), since a terminal or editor
// redraws the same runes over and over. The cache is keyed on the rune alone
// (not sub-pixel position): Face always rasterizes at an integer-pixel
// origin, so the same rune always produces the same mask — correct and
// simple for a monospace grid, at the cost of not doing sub-pixel-accurate
// text layout (irrelevant for cell-based UIs).
package ttf

import (
	"fmt"
	"image"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/X11Libre/go-x11proto/proto/base"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_render "github.com/X11Libre/go-x11proto/tk/render"
)

// Face is an antialiased text renderer for one TrueType/OpenType font at one
// size. It must be closed with Close to release cached X server resources.
type Face struct {
	conn *tk_core.TkConn
	rdr  *tk_render.Render
	face font.Face

	// parsed/dpi are kept so Resize can re-rasterize at a new point size
	// without re-reading the font file.
	parsed *opentype.Font
	size   float64
	dpi    float64

	a8   tk_render.PICTFORMAT
	argb tk_render.PICTFORMAT
	root base.DRAWABLE
	gc8  *tk_core.GC // depth-8 GC, valid for any depth-8 pixmap on this screen

	glyphs  map[rune]*glyphEntry
	sources map[base.CARD32]*colorSource
}

type glyphEntry struct {
	pic           *tk_render.Picture
	pixmap        *tk_core.Pixmap
	offX, offY    int // dr.Min, relative to the glyph's zero-position dot
	width, height int
	advance       int
	empty         bool // space, combining marks, or a glyph with no ink: nothing to composite
}

type colorSource struct {
	pic *tk_render.Picture
	pix *tk_core.Pixmap
}

// Open loads a TTF/OTF font file from path and rasterizes it at the given
// point size and DPI (typically the desktop's; see tk/theme).
func Open(conn *tk_core.TkConn, rdr *tk_render.Render, path string, size, dpi float64) (*Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ttf: read %s: %w", path, err)
	}
	parsed, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("ttf: parse %s: %w", path, err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    size,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("ttf: new face: %w", err)
	}

	a8, err := rdr.StandardFormat(8, true)
	if err != nil {
		return nil, fmt.Errorf("ttf: no A8 picture format: %w", err)
	}
	argb, err := rdr.ARGB32()
	if err != nil {
		return nil, fmt.Errorf("ttf: no ARGB32 picture format: %w", err)
	}
	root := base.DRAWABLE(conn.X11Conn.DefaultRoot())

	// A throwaway 1x1 depth-8 pixmap purely to mint a depth-8 GC: X requires
	// a GC's depth to match its drawable's, but once created the GC is a
	// standalone resource usable with any depth-8 drawable on this screen
	// (same root + same depth is the actual rule, not the same drawable).
	seed, err := conn.CreatePixmap(8, root, 1, 1)
	if err != nil {
		return nil, fmt.Errorf("ttf: seed pixmap: %w", err)
	}
	defer seed.Free()
	gc8, err := conn.CreateGCFor(seed.XID, 0, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("ttf: depth-8 GC: %w", err)
	}

	return &Face{
		conn: conn, rdr: rdr, face: face,
		parsed: parsed, size: size, dpi: dpi,
		a8: a8, argb: argb, root: root, gc8: gc8,
		glyphs:  make(map[rune]*glyphEntry),
		sources: make(map[base.CARD32]*colorSource),
	}, nil
}

// Size returns the current point size the face rasterizes at.
func (f *Face) Size() float64 { return f.size }

// Resize re-rasterizes the font at a new point size, freeing the cached
// glyph masks (which are size-specific) while keeping the colour sources.
// It returns an error only if the underlying font cannot be rebuilt; the
// previous size remains in effect then.
func (f *Face) Resize(size float64) error {
	for _, ge := range f.glyphs {
		if ge.pic != nil {
			_ = ge.pic.Free()
		}
		if ge.pixmap != nil {
			_ = ge.pixmap.Free()
		}
	}
	f.glyphs = make(map[rune]*glyphEntry)

	nf, err := opentype.NewFace(f.parsed, &opentype.FaceOptions{
		Size:    size,
		DPI:     f.dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	f.face = nf
	f.size = size
	return nil
}

// Height returns the recommended line height in whole pixels.
func (f *Face) Height() int { return f.face.Metrics().Height.Round() }

// Ascent returns the baseline's distance from the top of a line, in whole pixels.
func (f *Face) Ascent() int { return f.face.Metrics().Ascent.Round() }

// Advance returns r's advance width in whole pixels.
func (f *Face) Advance(r rune) int {
	ge, err := f.glyphFor(r)
	if err != nil {
		return 0
	}
	return ge.advance
}

// DrawString draws s with its baseline's left edge at (x, y) in dstPic,
// filled with color fg (opaque), returning the pen's final x position.
func (f *Face) DrawString(dstPic *tk_render.Picture, x, y int, s string, fg [3]byte) (int, error) {
	src, err := f.sourceFor(fg)
	if err != nil {
		return 0, err
	}
	cellH := f.Height()
	ascent := f.Ascent()
	penX := x
	for _, r := range s {
		// Box-drawing runes are hand-drawn against the cell rectangle
		// instead of rasterized from the font — see boxdraw.go for why.
		if arms, isBox := boxDrawTable[r]; isBox {
			adv, ok := f.face.GlyphAdvance(r)
			if ok {
				cellW := adv.Round()
				// cell top = y - ascent, matching where the font itself
				// would place a full-height glyph relative to baseline y.
				if err := drawBoxChar(dstPic, penX, y-ascent, cellW, cellH, arms, fg); err != nil {
					return 0, err
				}
				penX += cellW
				continue
			}
		}
		ge, err := f.glyphFor(r)
		if err != nil {
			return 0, err
		}
		if !ge.empty {
			if err := dstPic.Composite(tk_render.OpOver, src, ge.pic,
				0, 0, 0, 0, base.INT16(penX+ge.offX), base.INT16(y+ge.offY),
				base.CARD16(ge.width), base.CARD16(ge.height)); err != nil {
				return 0, err
			}
		}
		penX += ge.advance
	}
	return penX, nil
}

// glyphFor rasterizes r on first use and caches the result. Rasterization
// always happens at the zero dot (0, 0): since that has no fractional
// sub-pixel part, the same rune always yields the same mask, so caching by
// rune alone is correct for integer-pixel-positioned (cell-grid) text.
func (f *Face) glyphFor(r rune) (*glyphEntry, error) {
	if ge, ok := f.glyphs[r]; ok {
		return ge, nil
	}
	dot := fixed.Point26_6{}
	dr, mask, maskp, advance, ok := f.face.Glyph(dot, r)
	advPixels := advance.Round()
	if !ok || dr.Empty() {
		ge := &glyphEntry{empty: true, advance: advPixels}
		f.glyphs[r] = ge
		return ge, nil
	}
	alpha, isAlpha := mask.(*image.Alpha)
	if !isAlpha {
		return nil, fmt.Errorf("ttf: unsupported glyph mask type %T for %q", mask, r)
	}

	w, h := dr.Dx(), dr.Dy()
	scanline := (w + 3) / 4 * 4 // PutImage ZPixmap rows pad to 4 bytes regardless of depth
	data := make([]byte, scanline*h)
	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			data[row*scanline+col] = alpha.AlphaAt(maskp.X+col, maskp.Y+row).A
		}
	}

	pixmap, err := f.conn.CreatePixmap(8, f.root, base.CARD16(w), base.CARD16(h))
	if err != nil {
		return nil, err
	}
	if err := pixmap.Drawable.PutImage(f.gc8.XID, 2, 8, base.CARD16(w), base.CARD16(h), data); err != nil {
		_ = pixmap.Free()
		return nil, err
	}
	pic, err := f.rdr.NewPicture(pixmap.XID, f.a8, tk_render.PictureValues{})
	if err != nil {
		_ = pixmap.Free()
		return nil, err
	}

	ge := &glyphEntry{pic: pic, pixmap: pixmap, offX: dr.Min.X, offY: dr.Min.Y, width: w, height: h, advance: advPixels}
	f.glyphs[r] = ge
	return ge, nil
}

// sourceFor returns the cached 1x1 solid-color repeated Picture for fg,
// creating it on first use. This is the standard pre-CreateSolidFill trick:
// a real toolkit would use RENDER's CreateSolidFill request directly, but
// this repo's RENDER binding doesn't implement that request, and a 1x1
// RepeatNormal pixmap is exactly equivalent for compositing purposes.
func (f *Face) sourceFor(fg [3]byte) (*tk_render.Picture, error) {
	key := base.CARD32(fg[0])<<16 | base.CARD32(fg[1])<<8 | base.CARD32(fg[2])
	if cs, ok := f.sources[key]; ok {
		return cs.pic, nil
	}
	pix, err := f.conn.CreatePixmap(32, f.root, 1, 1)
	if err != nil {
		return nil, err
	}
	gc32, err := f.conn.CreateGCFor(pix.XID, 0, 0, 0)
	if err != nil {
		_ = pix.Free()
		return nil, err
	}
	defer gc32.Free()
	pixel := packARGB32(f.conn.X11Conn.Setup.ImageByteOrder, 0xff, fg[0], fg[1], fg[2])
	if err := pix.Drawable.PutImage(gc32.XID, 2, 32, 1, 1, pixel); err != nil {
		_ = pix.Free()
		return nil, err
	}
	pic, err := f.rdr.NewPicture(pix.XID, f.argb, tk_render.PictureValues{
		ValueMask: tk_render.CPRepeat,
		Repeat:    tk_render.RepeatNormal,
	})
	if err != nil {
		_ = pix.Free()
		return nil, err
	}
	f.sources[key] = &colorSource{pic: pic, pix: pix}
	return pic, nil
}

// packARGB32 builds one ARGB32-format pixel (the RENDER "standard format":
// alpha shift 24, red 16, green 8, blue 0) as raw bytes in the connection's
// byte order, opaque (alpha 0xff premultiplied is a no-op at full alpha).
func packARGB32(byteOrder base.CARD8, a, r, g, b byte) []byte {
	if byteOrder == 0 { // LSBFirst: low-to-high byte = B,G,R,A
		return []byte{b, g, r, a}
	}
	return []byte{a, r, g, b} // MSBFirst: high-to-low byte = A,R,G,B
}

// Close releases every cached glyph mask, color source, and the depth-8 GC.
func (f *Face) Close() error {
	var firstErr error
	note := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, ge := range f.glyphs {
		if ge.pic != nil {
			note(ge.pic.Free())
		}
		if ge.pixmap != nil {
			note(ge.pixmap.Free())
		}
	}
	for _, cs := range f.sources {
		note(cs.pic.Free())
		note(cs.pix.Free())
	}
	note(f.gc8.Free())
	return firstErr
}
