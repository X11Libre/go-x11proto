package xpm

import (
	"image"
	"image/draw"
)

// FromImage converts any image.Image into an Image holding tightly-packed,
// straight-alpha RGBA pixel data (4 bytes per pixel), ready for Image.Upload.
//
// An origin-anchored, tightly-packed *image.NRGBA is adopted directly (its Pix
// buffer is reused, no copy); any other type or layout (sub-image, RGBA,
// paletted, non-zero origin, padded stride) is normalised through a fresh NRGBA
// buffer so the result is always correct.
func FromImage(src image.Image) *Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if n, ok := src.(*image.NRGBA); ok && b.Min.X == 0 && b.Min.Y == 0 && n.Stride == w*4 {
		return &Image{Width: w, Height: h, Data: n.Pix}
	}
	rgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	return &Image{Width: w, Height: h, Data: rgba.Pix}
}
