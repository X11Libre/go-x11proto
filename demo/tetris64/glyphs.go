package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
	"github.com/X11Libre/go-x11proto/tk/xpm"
)

// High-resolution greyscale master glyphs (the manually-extracted originals).
// These are the source for every resolution: they are scaled down for 320x200
// and up for the larger modes, preserving their anti-aliased grey pixels rather
// than being reduced to a 1-bit bitmap.
//
//go:embed assets/320x200-mono/orig/*.png
var glyphMastersFS embed.FS

type glyphMaster struct {
	img                    image.Image
	minX, minY, maxX, maxY int // content bounding box (non-black pixels)
}

var glyphMasters [10]*glyphMaster

func glyphLum(img image.Image, x, y int) int {
	r, g, b, _ := img.At(x, y).RGBA()
	return (int(r>>8)*30 + int(g>>8)*59 + int(b>>8)*11) / 100
}

// loadGlyphMasters decodes the 0..9 master PNGs once and records each glyph's
// content bounding box. On-disk files (editable without a rebuild) take
// precedence over the embedded copies.
func loadGlyphMasters() {
	for d := 0; d < 10; d++ {
		if glyphMasters[d] != nil {
			continue
		}
		var data []byte
		if p := assetPathFor(fmt.Sprintf("320x200-mono/orig/%d.png", d)); p != "" {
			data, _ = os.ReadFile(p)
		}
		if data == nil {
			data, _ = glyphMastersFS.ReadFile(fmt.Sprintf("assets/320x200-mono/orig/%d.png", d))
		}
		if data == nil {
			continue
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}
		b := img.Bounds()
		minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
		found := false
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if glyphLum(img, x, y) >= 30 {
					found = true
					if x < minX {
						minX = x
					}
					if y < minY {
						minY = y
					}
					if x > maxX {
						maxX = x
					}
					if y > maxY {
						maxY = y
					}
				}
			}
		}
		if !found {
			continue
		}
		glyphMasters[d] = &glyphMaster{img, minX, minY, maxX, maxY}
	}
}

// buildDigitPixmaps renders each digit master into a greyscale (8*scale)^2 cell
// pixmap sized for the current resolution. The glyph is scaled (area-averaged)
// to 7 C64 rows tall with proportional width, centred horizontally, with a 1
// C64-pixel top margin — i.e. the same metrics as the bitmap font, but greyscale.
func (w *TetrisWin) buildDigitPixmaps() {
	loadGlyphMasters()
	// release any pixmaps from a previous resolution
	for d := range w.digitPix {
		if w.digitPix[d] != 0 {
			w.conn.Send(&freePixmapReq{pixmap: w.digitPix[d]})
			w.digitPix[d] = 0
		}
	}
	scale := w.scale
	cell := 8 * scale
	gh := 7 * scale  // glyph height: 7 C64 rows
	top := scale     // 1 C64-pixel top margin
	for d := 0; d < 10; d++ {
		m := glyphMasters[d]
		if m == nil {
			w.digitPix[d] = 0
			continue
		}
		bw := m.maxX - m.minX + 1
		bh := m.maxY - m.minY + 1
		gw := bw * gh / bh // keep each glyph's aspect (so e.g. '4' stays wider)
		if gw > cell {
			gw = cell
		}
		xoff := (cell - gw) / 2

		px := make([]byte, cell*cell*4)
		for i := 3; i < len(px); i += 4 {
			px[i] = 0xFF // opaque black background
		}
		for ty := 0; ty < gh; ty++ {
			for tx := 0; tx < gw; tx++ {
				// source region covered by this destination pixel
				sx0 := m.minX + tx*bw/gw
				sx1 := m.minX + (tx+1)*bw/gw
				sy0 := m.minY + ty*bh/gh
				sy1 := m.minY + (ty+1)*bh/gh
				if sx1 <= sx0 {
					sx1 = sx0 + 1
				}
				if sy1 <= sy0 {
					sy1 = sy0 + 1
				}
				var sr, sg, sb, n int
				for yy := sy0; yy < sy1; yy++ {
					for xx := sx0; xx < sx1; xx++ {
						r, g, b, _ := m.img.At(xx, yy).RGBA()
						sr += int(r >> 8)
						sg += int(g >> 8)
						sb += int(b >> 8)
						n++
					}
				}
				if n == 0 {
					n = 1
				}
				o := ((top+ty)*cell + xoff + tx) * 4
				px[o] = byte(sr / n)
				px[o+1] = byte(sg / n)
				px[o+2] = byte(sb / n)
				px[o+3] = 0xFF
			}
		}
		img := &xpm.Image{Width: cell, Height: cell, Data: px}
		pm, err := img.Upload(w.conn, w.conn.DefaultRoot())
		if err != nil {
			w.digitPix[d] = 0
			continue
		}
		w.digitPix[d] = pm
	}
}

// drawNumber blits the greyscale digit pixmaps for text onto target, advancing
// one fixed-width cell (8*scale) per character.
func (w *TetrisWin) drawNumber(target base.DRAWABLE, x, y base.INT16, text string) {
	cell := 8 * w.scale
	for i, ch := range text {
		if ch < '0' || ch > '9' {
			continue
		}
		pm := w.digitPix[ch-'0']
		if pm == 0 {
			continue
		}
		w.conn.Send(&copyAreaReq{
			src: pm, dst: target, gc: w.gcText,
			dstX:   x + base.INT16(i*cell),
			dstY:   y,
			width:  base.CARD16(cell),
			height: base.CARD16(cell),
		})
	}
}

// ---- raw CopyArea request (no rpc helper exists) ----

type copyAreaReq struct {
	src, dst       base.DRAWABLE
	gc             base.GC
	srcX, srcY     base.INT16
	dstX, dstY     base.INT16
	width, height  base.CARD16
}

func (r *copyAreaReq) WriteInto(wr *base.RequestWriter) error {
	wr.SetOpcode(opcode.CopyArea)
	wr.WriteCARD32(base.CARD32(r.src))
	wr.WriteCARD32(base.CARD32(r.dst))
	wr.WriteCARD32(base.CARD32(r.gc))
	wr.WriteINT16(r.srcX)
	wr.WriteINT16(r.srcY)
	wr.WriteINT16(r.dstX)
	wr.WriteINT16(r.dstY)
	wr.WriteCARD16(r.width)
	wr.WriteCARD16(r.height)
	return nil
}

type freePixmapReq struct{ pixmap base.PIXMAP }

func (r *freePixmapReq) WriteInto(wr *base.RequestWriter) error {
	wr.SetOpcode(opcode.FreePixmap)
	wr.WriteCARD32(base.CARD32(r.pixmap))
	return nil
}
