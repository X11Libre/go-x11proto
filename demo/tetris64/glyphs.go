package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"

	"github.com/X11Libre/go-x11proto/tk/xpm"
)

// Master digit glyphs: aligned, centred, baseline-matched, with a transparent
// (alpha) background. One shared set drives every theme/resolution — they are
// scaled per resolution and tinted per theme at load time. The alpha channel is
// the glyph coverage; the RGB (white) is irrelevant since the digits are tinted.
// The rendering itself lives in the font package (font.Digits); here we only
// load the master images and pick the tint colour from the background art.
//
//go:embed assets/glyphs/*.png
var glyphMastersFS embed.FS

var glyphMasters [10]image.Image

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
	const fhd = 0 // index of the FHD resolution / layout
	if img, err := decodeImage(loadFrame()); err == nil {
		t = sampleGlyphTint(img, layouts[fhd], resolutions[fhd].scale)
	}
	cachedGlyphTint = &t
	return t
}

var cachedBorder *[3]byte

// wellBorderColor samples the colour of the play-field border that is drawn
// into the background art (a thin lavender line in the colour theme), so the
// help page border can be made to match it. Cached, re-sampled per theme.
func wellBorderColor() [3]byte {
	if cachedBorder != nil {
		return *cachedBorder
	}
	c := [3]byte{0xBA, 0xBA, 0xBA}
	const fhd = 0
	if img, err := decodeImage(loadFrame()); err == nil {
		c = sampleBorder(img, layouts[fhd], resolutions[fhd].scale)
	}
	cachedBorder = &c
	return c
}

// sampleBorder averages the brighter pixels in a thin band along the top edge
// of the well, i.e. the play-field border line, ignoring the black interior.
func sampleBorder(img *xpm.Image, l resLayout, scale int) [3]byte {
	x0, x1 := l.bx, l.bx+10*l.cell
	y0, y1 := l.by-3*scale, l.by+2*scale
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
	at := func(x, y int) (int, int, int, int) {
		o := (y*img.Width + x) * 4
		r, g, b := int(img.Data[o]), int(img.Data[o+1]), int(img.Data[o+2])
		return r, g, b, (r*30 + g*59 + b*11) / 100
	}
	maxLum := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if _, _, _, lum := at(x, y); lum > maxLum {
				maxLum = lum
			}
		}
	}
	if maxLum < 40 {
		return [3]byte{0xBA, 0xBA, 0xBA}
	}
	var sr, sg, sb, n int
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, b, lum := at(x, y)
			if lum*5 >= maxLum*2 { // brighter half of the line, skips the black gap
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

// sampleGlyphTint picks the score-digit colour out of the background art so the
// rendered digits match it: it scans the score/lines/level panel for the
// brightest text pixels and averages them. Falls back to a washed-out grey.
func sampleGlyphTint(img *xpm.Image, l resLayout, scale int) [3]byte {
	x0, y0 := l.scoreX, l.scoreY
	x1 := l.scoreX + 8*8*scale
	y1 := l.linesY + 8*scale
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

// loadGlyphMasters decodes the 0..9 embedded master PNGs once (cached) and
// returns them for use as a font.Digits backing set.
func loadGlyphMasters() [10]image.Image {
	for d := 0; d < 10; d++ {
		if glyphMasters[d] != nil {
			continue
		}
		data, _ := glyphMastersFS.ReadFile(fmt.Sprintf("assets/glyphs/%d.png", d))
		if data == nil {
			continue
		}
		if img, err := png.Decode(bytes.NewReader(data)); err == nil {
			glyphMasters[d] = img
		}
	}
	return glyphMasters
}
