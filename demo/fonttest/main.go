// Command fonttest is a proof of concept for antialiased TrueType text on
// go-x11proto: it rasterizes glyphs with golang.org/x/image/font/{sfnt,
// opentype} (pure Go, no cgo/libfreetype) and blits each glyph's alpha mask
// onto the window via the RENDER extension's Composite request (tk/render) —
// the same technique Xft uses, just with a Go rasterizer instead of C
// FreeType. It draws the same string twice, once with the existing X core
// bitmap font and once with the new antialiased path, so the difference is
// visible side by side. Standalone and throwaway: this is not wired into
// tk/font or tk/term yet, just validation that the rendering technique
// works before building a real TextRenderer backend on top of it.
package main

import (
	"fmt"
	"image"
	"log"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_font "github.com/X11Libre/go-x11proto/tk/font"
	tk_render "github.com/X11Libre/go-x11proto/tk/render"
)

// ttfPath is a system-installed monospace font, good enough for this
// throwaway comparison; a real backend will need embedding or fontconfig
// discovery instead of a hardcoded path.
const ttfPath = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"

const sampleText = "The quick Fox 0123 ┌─┐ ╔═╗"

// aaGlyphRenderer draws antialiased glyphs from an x/image font.Face onto a
// RENDER Picture by rasterizing each glyph to an alpha mask, uploading it as
// an 8-bit-depth pixmap, and compositing it with a solid-color source —
// exactly what Xft does internally, minus the C library.
type aaGlyphRenderer struct {
	conn   *tk_core.TkConn
	rdr    *tk_render.Render
	face   font.Face
	a8     tk_render.PICTFORMAT
	srcPic *tk_render.Picture // 1x1 solid-color source, repeated
	srcPix *tk_core.Pixmap
	gc8    *tk_core.GC // depth-8 GC, reusable for any depth-8 pixmap (X only requires matching root+depth, not the same drawable)
	root   base.DRAWABLE
}

func newAAGlyphRenderer(conn *tk_core.TkConn, rdr *tk_render.Render, face font.Face, fg [3]byte) (*aaGlyphRenderer, error) {
	a8, err := rdr.StandardFormat(8, true)
	if err != nil {
		return nil, fmt.Errorf("no A8 picture format: %w", err)
	}
	argb, err := rdr.ARGB32()
	if err != nil {
		return nil, fmt.Errorf("no ARGB32 picture format: %w", err)
	}
	root := base.DRAWABLE(conn.X11Conn.DefaultRoot())

	// A GC's depth must match its drawable's (X11 core protocol requirement:
	// same root + same depth), so the root-depth GC from CreateGC1 cannot be
	// used to PutImage into an 8-bit alpha mask or a 32-bit ARGB pixmap —
	// each needs its own GC created against a same-depth drawable.
	srcPix, err := conn.CreatePixmap(32, root, 1, 1)
	if err != nil {
		return nil, err
	}
	gc32, err := conn.CreateGCFor(srcPix.XID, conn.X11Conn.DefaultBlackPixel(), conn.X11Conn.DefaultWhitePixel(), 0)
	if err != nil {
		return nil, err
	}
	defer gc32.Free()

	pixel := packARGB32(conn.X11Conn.Setup.ImageByteOrder, 0xff, fg[0], fg[1], fg[2])
	if err := srcPix.Drawable.PutImage(gc32.XID, 2, 32, 1, 1, pixel); err != nil {
		return nil, err
	}
	// 1x1 opaque solid-color source picture, tiled via RepeatNormal — the
	// classic pre-CreateSolidFill trick: a real toolkit would use the
	// RENDER CreateSolidFill request directly, but this repo's RENDER
	// binding doesn't implement that request (yet), and a 1x1 repeated
	// pixmap is exactly equivalent.
	srcPic, err := rdr.NewPicture(srcPix.XID, argb, tk_render.PictureValues{
		ValueMask: tk_render.CPRepeat,
		Repeat:    tk_render.RepeatNormal,
	})
	if err != nil {
		return nil, err
	}

	glyphPix, err := conn.CreatePixmap(8, root, 1, 1)
	if err != nil {
		return nil, err
	}
	defer glyphPix.Free()
	gc8, err := conn.CreateGCFor(glyphPix.XID, 0, 0, 0)
	if err != nil {
		return nil, err
	}

	return &aaGlyphRenderer{conn: conn, rdr: rdr, face: face, a8: a8, srcPic: srcPic, srcPix: srcPix, gc8: gc8, root: root}, nil
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

// DrawString draws s with its baseline starting at (x, y) in dst's picture,
// returning the pen's final x position.
func (a *aaGlyphRenderer) DrawString(dstPic *tk_render.Picture, x, y int, s string) (int, error) {
	dot := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
	for _, r := range s {
		dr, mask, maskp, advance, ok := a.face.Glyph(dot, r)
		if ok && !dr.Empty() {
			if err := a.blit(dstPic, dr, mask, maskp); err != nil {
				return 0, err
			}
		}
		dot.X += advance
	}
	return dot.X.Round(), nil
}

func (a *aaGlyphRenderer) blit(dstPic *tk_render.Picture, dr image.Rectangle, mask image.Image, maskp image.Point) error {
	alpha, ok := mask.(*image.Alpha)
	if !ok {
		return fmt.Errorf("unsupported glyph mask type %T", mask)
	}
	w, h := dr.Dx(), dr.Dy()
	scanline := (w + 3) / 4 * 4 // RENDER/PutImage ZPixmap rows pad to 4 bytes regardless of depth
	data := make([]byte, scanline*h)
	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			data[row*scanline+col] = alpha.AlphaAt(maskp.X+col, maskp.Y+row).A
		}
	}

	pixmap, err := a.conn.CreatePixmap(8, a.root, base.CARD16(w), base.CARD16(h))
	if err != nil {
		return err
	}
	defer pixmap.Free()
	if err := pixmap.Drawable.PutImage(a.gc8.XID, 2, 8, base.CARD16(w), base.CARD16(h), data); err != nil {
		return err
	}
	maskPic, err := a.rdr.NewPicture(pixmap.XID, a.a8, tk_render.PictureValues{})
	if err != nil {
		return err
	}
	defer maskPic.Free()

	return dstPic.Composite(tk_render.OpOver, a.srcPic, maskPic,
		0, 0, 0, 0, base.INT16(dr.Min.X), base.INT16(dr.Min.Y),
		base.CARD16(w), base.CARD16(h))
}

type win struct {
	tk_core.Window
	rdr      *tk_render.Render
	winPic   *tk_render.Picture
	coreFont *tk_font.Font
	coreGC   *tk_core.GC
	aa       *aaGlyphRenderer
}

func (w *win) HandleWindowEvent(ev events.Event) bool {
	if _, ok := ev.(*events.ExposeEvent); ok {
		_ = w.draw()
	}
	return true
}

func (w *win) draw() error {
	if err := w.Window.FillRect(w.coreGC.XID, 0, 0, 600, 160); err != nil {
		return err
	}
	if err := w.coreFont.SetOn(w.coreGC); err != nil {
		return err
	}
	if err := w.Window.Drawable.ImageText8(w.coreGC.XID, 10, 40, "core bitmap font: "+sampleText); err != nil {
		return err
	}
	_, err := w.aa.DrawString(w.winPic, 10, 110, "AA TrueType font: "+sampleText)
	return err
}

func main() {
	conn, err := proto.DialBE("")
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()
	tkc := tk_core.MakeTkConn(conn)

	rdr, err := tk_render.Open(&tkc)
	if err != nil {
		log.Fatalf("RENDER extension unavailable: %v", err)
	}

	ttfData, err := os.ReadFile(ttfPath)
	if err != nil {
		log.Fatalf("read %s: %v", ttfPath, err)
	}
	parsed, err := opentype.Parse(ttfData)
	if err != nil {
		log.Fatalf("parse ttf: %v", err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    18,
		DPI:     96,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf("new face: %v", err)
	}

	coreFont, err := tk_font.Open(conn, "fixed")
	if err != nil {
		log.Fatalf("open core font: %v", err)
	}
	coreGC, err := tkc.CreateGC1(tkc.X11Conn.DefaultBlackPixel(), tkc.X11Conn.DefaultWhitePixel(), 0)
	if err != nil {
		log.Fatalf("create GC: %v", err)
	}

	w := &win{
		Window: tk_core.Window{
			Drawable:  tk_core.Drawable{Conn: &tkc},
			Parent:    tkc.GetRoot(),
			Name:      "fonttest: core bitmap vs antialiased TrueType",
			X:         50,
			Y:         50,
			W:         600,
			H:         160,
			EventMask: 0x8000, // ExposureMask
		},
		rdr:      rdr,
		coreFont: coreFont,
		coreGC:   coreGC,
	}
	w.Window.SetWindowHandler(w)
	if err := w.Window.Create(); err != nil {
		log.Fatalf("create window: %v", err)
	}

	// CreatePicture (like CreateWindow/CreatePixmap) allocates its XID
	// client-side and has no reply, so a format/depth mismatch surfaces only
	// as an async X error, never as this call's return value — the format
	// must be picked correctly up front from the window's real depth, not
	// guessed and "corrected" after the fact.
	geom, err := w.Window.GetGeometry()
	if err != nil {
		log.Fatalf("GetGeometry: %v", err)
	}
	winFmt, err := rdr.StandardFormat(geom.Depth, false)
	if err != nil {
		log.Fatalf("no picture format for window depth %d: %v", geom.Depth, err)
	}
	w.winPic, err = rdr.PictureFor(w.Window.Drawable, winFmt, tk_render.PictureValues{})
	if err != nil {
		log.Fatalf("picture for window: %v", err)
	}

	aa, err := newAAGlyphRenderer(&tkc, rdr, face, [3]byte{0x10, 0x10, 0x10})
	if err != nil {
		log.Fatalf("aa glyph renderer: %v", err)
	}
	w.aa = aa

	w.Window.Map()

	conn.SimpleEventLoop()
}
