package core

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/request"
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
	if len(rects) == 0 {
		return nil // nothing to draw: skip the round-trip
	}
	return rpc.FillRects(d.Conn.X11Conn, d.XID, gc, rects)
}

func (d Drawable) PutText8(gc base.GC, x base.INT16, y base.INT16, text string) error {
	return rpc.PutText8(d.Conn.X11Conn, d.XID, gc, x, y, text)
}

func (d Drawable) PutImage(gc base.GC, format base.CARD8, depth base.CARD8, width base.CARD16, height base.CARD16, data []byte) error {
	return rpc.PutImage(d.Conn.X11Conn, d.XID, gc, format, depth, width, height, 0, 0, data)
}

// ---- copy operations (d is the source) ----

func (d Drawable) CopyArea(dst base.DRAWABLE, gc base.GC, srcX, srcY, dstX, dstY base.INT16, width, height base.CARD16) error {
	return rpc.CopyArea(d.Conn.X11Conn, d.XID, dst, gc, srcX, srcY, dstX, dstY, width, height)
}

func (d Drawable) CopyPlane(dst base.DRAWABLE, gc base.GC, srcX, srcY, dstX, dstY base.INT16, width, height base.CARD16, bitPlane base.CARD32) error {
	return rpc.CopyPlane(d.Conn.X11Conn, d.XID, dst, gc, srcX, srcY, dstX, dstY, width, height, bitPlane)
}

// ---- drawing primitives ----

func (d Drawable) PolyPoint(gc base.GC, coordMode base.CARD8, points []base.Point) error {
	return rpc.PolyPoint(d.Conn.X11Conn, coordMode, d.XID, gc, points)
}

func (d Drawable) PolyLine(gc base.GC, coordMode base.CARD8, points []base.Point) error {
	return rpc.PolyLine(d.Conn.X11Conn, coordMode, d.XID, gc, points)
}

func (d Drawable) PolySegment(gc base.GC, segments []base.Segment) error {
	return rpc.PolySegment(d.Conn.X11Conn, d.XID, gc, segments)
}

func (d Drawable) PolyRectangle(gc base.GC, rects []base.Rectangle) error {
	return rpc.PolyRectangle(d.Conn.X11Conn, d.XID, gc, rects)
}

func (d Drawable) PolyArc(gc base.GC, arcs []base.Arc) error {
	return rpc.PolyArc(d.Conn.X11Conn, d.XID, gc, arcs)
}

func (d Drawable) FillPoly(gc base.GC, shape, coordMode base.CARD8, points []base.Point) error {
	return rpc.FillPoly(d.Conn.X11Conn, d.XID, gc, shape, coordMode, points)
}

func (d Drawable) PolyFillArc(gc base.GC, arcs []base.Arc) error {
	return rpc.PolyFillArc(d.Conn.X11Conn, d.XID, gc, arcs)
}

// ---- text ----

func (d Drawable) ImageText8(gc base.GC, x, y base.INT16, text string) error {
	return rpc.ImageText8(d.Conn.X11Conn, d.XID, gc, x, y, text)
}

func (d Drawable) ImageText16(gc base.GC, x, y base.INT16, text []base.CARD16) error {
	return rpc.ImageText16(d.Conn.X11Conn, d.XID, gc, x, y, text)
}

func (d Drawable) PolyText16(gc base.GC, x, y base.INT16, text []base.CARD16) error {
	return rpc.PolyText16(d.Conn.X11Conn, d.XID, gc, x, y, text)
}

// ---- queries ----

func (d Drawable) GetGeometry() (*request.GetGeometryReply, error) {
	return rpc.GetGeometry(d.Conn.X11Conn, d.XID)
}

func (d Drawable) GetImage(format base.CARD8, x, y base.INT16, width, height base.CARD16, planeMask base.CARD32) (*request.GetImageReply, error) {
	return rpc.GetImage(d.Conn.X11Conn, format, d.XID, x, y, width, height, planeMask)
}
