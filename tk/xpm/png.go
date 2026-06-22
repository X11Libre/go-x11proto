package xpm

import (
	"bytes"
	"image/png"
)

// DecodePNG decodes PNG bytes into an Image with straight-alpha (NRGBA) pixel
// data, ready to be uploaded as a pixmap via Image.Upload.
func DecodePNG(data []byte) (*Image, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return FromImage(img), nil
}
