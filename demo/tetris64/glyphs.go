package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	"github.com/X11Libre/go-x11proto/tk/xpm"
)

// Master digit glyphs: aligned, centred, baseline-matched, with a transparent
// (alpha) background. One shared set drives every theme/resolution — they are
// scaled per resolution and tinted per theme at load time. The alpha channel is
// the glyph coverage; the RGB (white) is irrelevant since the digits are tinted.
//
//go:embed assets/glyph-masters/*.png
var glyphMastersFS embed.FS

var glyphMasters [10]image.Image

// glyphLumFull is the coverage value treated as "fully lit" (the masters' peak
// alpha): coverage at or above it maps to the full tint colour.
const glyphLumFull = 205

// tintByte scales tint channel t by coverage cov (0..glyphLumFull -> 0..t).
func tintByte(t byte, cov int) byte {
	v := int(t) * cov / glyphLumFull
	if v > 255 {
		v = 255
	}
	return byte(v)
}

var cachedGlyphTint *[3]byte

// glyphTintColor returns the digit colour to match the background art. It is
// sampled from the FHD frame (where the baked-in score text is sharp and at
// full brightness) and reused for every resolution, so the digits keep a
// consistent colour instead of fading with the downscaled small-resolution art.
func glyphTintColor() [3]byte {
	if cachedGlyphTint != nil {
		return *cachedGlyphTint
	}
	t := [3]byte{0xBA, 0xBA, 0xBA}
	const fhd = 1 // index of the FHD resolution / layout
	if img, err := decodeImage(loadFrame(resolutions[fhd].name)); err == nil {
		t = sampleGlyphTint(img, layouts[fhd], resolutions[fhd].scale)
	}
	cachedGlyphTint = &t
	return t
}

// sampleGlyphTint picks the score-digit colour out of the background art so the
// rendered digits match it: it scans the score/lines/level panel for the
// brightest text pixels and averages them. Falls back to a washed-out grey.
func sampleGlyphTint(img *xpm.Image, l resLayout, scale int) [3]byte {
	x0, y0 := l.scoreX, l.scoreY
	x1 := l.scoreX + 8*8*scale
	y1 := l.levelY + 8*scale
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > img.Width {
		x1 = img.Width
	}
	if y1 > img.Height {
		y1 = img.Height
	}
	maxLum := 0
	at := func(x, y int) (int, int, int, int) {
		o := (y*img.Width + x) * 4
		r, g, b := int(img.Data[o]), int(img.Data[o+1]), int(img.Data[o+2])
		return r, g, b, (r*30 + g*59 + b*11) / 100
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if _, _, _, lum := at(x, y); lum > maxLum {
				maxLum = lum
			}
		}
	}
	if maxLum < 60 {
		return [3]byte{0xBA, 0xBA, 0xBA} // no clear text found; washed-out grey
	}
	var sr, sg, sb, n int
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, b, lum := at(x, y)
			if lum >= maxLum-24 {
				sr += r
				sg += g
				sb += b
				n++
			}
		}
	}
	if n == 0 {
		return [3]byte{0xBA, 0xBA, 0xBA}
	}
	return [3]byte{byte(sr / n), byte(sg / n), byte(sb / n)}
}

// loadGlyphMasters decodes the 0..9 master PNGs once. On-disk files (editable
// without a rebuild) take precedence over the embedded copies.
func loadGlyphMasters() {
	for d := 0; d < 10; d++ {
		if glyphMasters[d] != nil {
			continue
		}
		var data []byte
		if p := assetPathFor(fmt.Sprintf("glyph-masters/%d.png", d)); p != "" {
			data, _ = os.ReadFile(p)
		}
		if data == nil {
			data, _ = glyphMastersFS.ReadFile(fmt.Sprintf("assets/glyph-masters/%d.png", d))
		}
		if data == nil {
			continue
		}
		if img, err := png.Decode(bytes.NewReader(data)); err == nil {
			glyphMasters[d] = img
		}
	}
}

// buildDigitPixmaps scales each (already aligned) master into an (8*scale)^2
// cell pixmap for the current resolution. The master's alpha channel is the
// coverage; it is area-averaged down/up to the cell size and composited over
// black in the tint colour (the pixmaps are opaque, blitted with CopyArea).
func (w *TetrisWin) buildDigitPixmaps() {
	loadGlyphMasters()
	// release any pixmaps from a previous resolution
	for d := range w.digitPix {
		if w.digitPix[d] != 0 {
			rpc.FreePixmap(w.conn, w.digitPix[d])
			w.digitPix[d] = 0
		}
	}
	cell := 8 * w.scale
	for d := 0; d < 10; d++ {
		m := glyphMasters[d]
		if m == nil {
			w.digitPix[d] = 0
			continue
		}
		mb := m.Bounds()
		mw, mh := mb.Dx(), mb.Dy()
		px := make([]byte, cell*cell*4)
		for i := 3; i < len(px); i += 4 {
			px[i] = 0xFF // opaque black background
		}
		for ty := 0; ty < cell; ty++ {
			for tx := 0; tx < cell; tx++ {
				// source region of the master covered by this cell pixel
				sx0 := tx * mw / cell
				sx1 := (tx + 1) * mw / cell
				sy0 := ty * mh / cell
				sy1 := (ty + 1) * mh / cell
				if sx1 <= sx0 {
					sx1 = sx0 + 1
				}
				if sy1 <= sy0 {
					sy1 = sy0 + 1
				}
				var sa, n int
				for yy := sy0; yy < sy1; yy++ {
					for xx := sx0; xx < sx1; xx++ {
						_, _, _, a := m.At(mb.Min.X+xx, mb.Min.Y+yy).RGBA()
						sa += int(a >> 8)
						n++
					}
				}
				if n == 0 {
					n = 1
				}
				cov := sa / n // averaged alpha = glyph coverage
				o := (ty*cell + tx) * 4
				px[o] = tintByte(w.glyphTint[0], cov)
				px[o+1] = tintByte(w.glyphTint[1], cov)
				px[o+2] = tintByte(w.glyphTint[2], cov)
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

// drawNumber blits the digit pixmaps for text onto target, advancing one
// fixed-width cell (8*scale) per character.
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
		rpc.CopyArea(w.conn, pm, target, w.gcText,
			0, 0, x+base.INT16(i*cell), y,
			base.CARD16(cell), base.CARD16(cell))
	}
}
