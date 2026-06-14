package font

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

func GlyphWidth(scale int) int {
	return 8 * scale
}

func GlyphHeight(scale int) int {
	return 8 * scale
}

func DrawString(d tk_core.Drawable, gc base.GC, x, y base.INT16, scale int, text string) error {
	if scale <= 0 {
		return nil
	}
	var rects []base.Rectangle
	xOff := int(x)
	for _, ch := range text {
		g, ok := Glyphs[ch]
		if !ok {
			g = Glyphs[' ']
		}
		for row := 0; row < 8; row++ {
			bits := g[row]
			if bits == 0 {
				continue
			}
			for col := 0; col < 8; col++ {
				if bits&(0x80>>col) != 0 {
					rx := base.INT16(xOff + col*scale)
					ry := base.INT16(int(y) + row*scale)
					rects = append(rects, base.Rectangle{
						X: rx, Y: ry,
						Width:  base.CARD16(scale),
						Height: base.CARD16(scale),
					})
				}
			}
		}
		xOff += 8 * scale
	}
	if len(rects) == 0 {
		return nil
	}
	return rpc.FillRects(d.Conn.X11Conn, d.XID, gc, rects)
}
