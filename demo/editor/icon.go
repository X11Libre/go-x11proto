package main

import (
	"image"
	"image/color"
	"image/draw"
)

// editorIcon draws a 48x48 "document" window icon: a white page with a dark
// border, a blue title bar and a few grey text lines, on a transparent
// background.
func editorIcon() image.Image {
	const s = 48
	img := image.NewNRGBA(image.Rect(0, 0, s, s))
	fill := func(x0, y0, x1, y1 int, c color.NRGBA) {
		draw.Draw(img, image.Rect(x0, y0, x1, y1), &image.Uniform{C: c}, image.Point{}, draw.Src)
	}
	var (
		border = color.NRGBA{0x30, 0x30, 0x38, 0xff}
		page   = color.NRGBA{0xff, 0xff, 0xff, 0xff}
		title  = color.NRGBA{0x2b, 0x6c, 0xb0, 0xff} // blue
		line   = color.NRGBA{0xa8, 0xa8, 0xb0, 0xff}
	)
	fill(9, 4, 39, 45, border) // page outline
	fill(10, 5, 38, 44, page)  // page body
	fill(10, 5, 38, 13, title) // title bar
	for i := 0; i < 5; i++ {   // text lines
		y := 18 + i*5
		fill(14, y, 34, y+2, line)
	}
	return img
}
