package font

import (
	"bytes"
	"image/png"
)

// LoadGlyphPNG decodes an up-to-8x8 PNG and installs it as the glyph for r,
// replacing the built-in bitmap. A pixel counts as "on" when it is both
// sufficiently opaque and bright, so the function accepts either a
// white-on-transparent 1-bit mask or a gray-on-black screen capture.
func LoadGlyphPNG(r rune, data []byte) error {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	b := img.Bounds()
	var g [8]byte
	for row := 0; row < 8 && row < b.Dy(); row++ {
		for col := 0; col < 8 && col < b.Dx(); col++ {
			cr, cg, cb, ca := img.At(b.Min.X+col, b.Min.Y+row).RGBA()
			a := int(ca >> 8)
			lum := (int(cr>>8)*30 + int(cg>>8)*59 + int(cb>>8)*11) / 100
			if a >= 128 && lum >= 96 {
				g[row] |= 0x80 >> col
			}
		}
	}
	Glyphs[r] = g
	return nil
}
