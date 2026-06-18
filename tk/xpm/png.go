package xpm

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
)

// DecodePNG decodes PNG bytes into an Image with straight-alpha (NRGBA) pixel
// data, ready to be uploaded as a pixmap via Image.Upload.
func DecodePNG(data []byte) (*Image, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	rgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	return &Image{Width: b.Dx(), Height: b.Dy(), Data: rgba.Pix}, nil
}
