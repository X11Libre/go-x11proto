package main

import (
	"bytes"
	"embed"
	"image"
	"image/png"

	xdraw "golang.org/x/image/draw"

	"github.com/X11Libre/go-x11proto/tk/xpm"
)

// Only the FHD background art is compiled in (both themes); WQHD and UHD4K are
// produced at load time by upscaling FHD (see uploadBg). This keeps the binary
// small at the cost of slightly softer background art on the higher resolutions
// — the live score/lines digits are rendered from masters at native scale, so
// they stay crisp regardless.
//
//go:embed assets/color assets/mono
var bgFS embed.FS

//go:embed assets/music/tetris.sid
var sidData []byte

// loadFrame / loadLoader return the embedded FHD background PNG for the current
// theme. Higher resolutions are upscaled from these at upload time.
func loadFrame() []byte {
	d, _ := bgFS.ReadFile("assets/" + theme + "/frame.png")
	return d
}

func loadLoader() []byte {
	d, _ := bgFS.ReadFile("assets/" + theme + "/loader.png")
	return d
}

// decodeImage decodes PNG (preferred) or XPM bytes into an xpm.Image.
func decodeImage(data []byte) (*xpm.Image, error) {
	if img, err := xpm.DecodePNG(data); err == nil {
		return img, nil
	}
	return xpm.DecodeBytes(data)
}

// decodeScaled decodes a PNG and returns it as an NRGBA-backed xpm.Image scaled
// to dstW x dstH. A same-size image is copied straight through; a larger target
// is interpolated with Catmull-Rom for a smooth upscale.
func decodeScaled(data []byte, dstW, dstH int) (*xpm.Image, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if src.Bounds().Dx() == dstW && src.Bounds().Dy() == dstH {
		return xpm.FromImage(src), nil // same size: just normalise to RGBA
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	return xpm.FromImage(dst), nil // fresh tight NRGBA: adopted without copying
}
