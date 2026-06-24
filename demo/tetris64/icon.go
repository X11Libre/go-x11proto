package main

import (
	"image"
	"image/color"
	"image/draw"
)

// tetrisIcon draws a 48x48 icon: a cyan T-tetromino (beveled cells) on a dark
// background — the C64-Tetris theme in miniature.
func tetrisIcon() image.Image {
	const s = 48
	img := image.NewNRGBA(image.Rect(0, 0, s, s))
	fill := func(x0, y0, x1, y1 int, c color.NRGBA) {
		draw.Draw(img, image.Rect(x0, y0, x1, y1), &image.Uniform{C: c}, image.Point{}, draw.Src)
	}
	var (
		bg    = color.NRGBA{0x10, 0x12, 0x20, 0xff}
		cyan  = color.NRGBA{0x18, 0xc0, 0xd0, 0xff}
		light = color.NRGBA{0x80, 0xf0, 0xff, 0xff} // top/left bevel
		dark  = color.NRGBA{0x0c, 0x70, 0x80, 0xff} // bottom/right bevel
	)
	fill(0, 0, s, s, bg)

	const cell = 11
	cellAt := func(col, row int) {
		x, y := 7+col*cell, 13+row*cell
		fill(x, y, x+cell, y+cell, dark)         // base / bottom-right
		fill(x, y, x+cell-1, y+cell-1, light)    // top-left bevel
		fill(x+1, y+1, x+cell-1, y+cell-1, cyan) // face
	}
	// a T-piece: three across, one below the centre
	cellAt(0, 0)
	cellAt(1, 0)
	cellAt(2, 0)
	cellAt(1, 1)
	return img
}
