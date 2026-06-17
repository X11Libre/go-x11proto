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

// High-resolution greyscale master glyphs (the manually-extracted originals).
// These are the source for every resolution: they are scaled down for 320x200
// and up for the larger modes, preserving their anti-aliased grey pixels rather
// than being reduced to a 1-bit bitmap.
//
// High-res greyscale digit masters, shared by all themes/resolutions: they are
// tinted (per theme) and scaled (per resolution) at runtime, so they are not
// split by theme/resolution like the backgrounds.
//
//go:embed assets/glyphs/*.png
var glyphMastersFS embed.FS

type glyphMaster struct {
	img                    image.Image
	minX, minY, maxX, maxY int // content bounding box (non-black pixels)
}

var glyphMasters [10]*glyphMaster

// glyphLumFull is the master grey level treated as "fully lit"; master pixels
// at or above it map to the full tint colour.
const glyphLumFull = 205

func glyphLum(img image.Image, x, y int) int {
	r, g, b, _ := img.At(x, y).RGBA()
	return (int(r>>8)*30 + int(g>>8)*59 + int(b>>8)*11) / 100
}

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

// loadGlyphMasters decodes the 0..9 master PNGs once and records each glyph's
// content bounding box. On-disk files (editable without a rebuild) take
// precedence over the embedded copies.
func loadGlyphMasters() {
	for d := 0; d < 10; d++ {
		if glyphMasters[d] != nil {
			continue
		}
		var data []byte
		if p := assetPathFor(fmt.Sprintf("glyphs/%d.png", d)); p != "" {
			data, _ = os.ReadFile(p)
		}
		if data == nil {
			data, _ = glyphMastersFS.ReadFile(fmt.Sprintf("assets/glyphs/%d.png", d))
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
			rpc.FreePixmap(w.conn, w.digitPix[d])
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
				// master luminance acts as coverage; fill with the tint colour
				cov := (sr/n*30 + sg/n*59 + sb/n*11) / 100
				o := ((top+ty)*cell + xoff + tx) * 4
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
		rpc.CopyArea(w.conn, pm, target, w.gcText,
			0, 0, x+base.INT16(i*cell), y,
			base.CARD16(cell), base.CARD16(cell))
	}
}
