package core

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

type Drawable struct {
	Conn *TkConn
	XID  base.DRAWABLE
}

func (d Drawable) Invalid() bool {
	return d.XID.Invalid()
}

func (d Drawable) FillRect(gc base.GC, x base.INT16, y base.INT16, width base.CARD16, height base.CARD16) error {
	return d.FillRects(
		gc,
		[]base.Rectangle{
			{X: x, Y: y, Width: width, Height: height},
		},
	)
}

func (d Drawable) FillRects(gc base.GC, rects []base.Rectangle) error {
	return rpc.FillRects(d.Conn.X11Conn, d.XID, gc, rects)
}

func (d Drawable) PutText8(gc base.GC, x base.INT16, y base.INT16, text string) error {
	return rpc.PutText8(d.Conn.X11Conn, d.XID, gc, x, y, text)
}

func (d Drawable) PutImage(gc base.GC, format base.CARD8, depth base.CARD8, width base.CARD16, height base.CARD16, data []byte) error {
	return rpc.PutImage(d.Conn.X11Conn, d.XID, gc, format, depth, width, height, 0, 0, data)
}
