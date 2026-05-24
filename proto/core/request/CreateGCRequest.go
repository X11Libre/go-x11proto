package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

const (
	GC_MASK_FUNCTION          = 0x0001
	GC_MASK_PLANE_MASK        = 0x0002
	GC_MASK_FOREGROUND        = 0x0004
	GC_MASK_BACKGROUND        = 0x0008
	GC_MASK_LINE_WIDTH        = 0x0010
	GC_MASK_LINE_STYLE        = 0x0020
	GC_MASK_CAP_STYLE         = 0x0040
	GC_MASK_JOIN_STYLE        = 0x0080
	GC_MASK_FILL_STYLE        = 0x0100
	GC_MASK_FILL_RULE         = 0x0200
	GC_MASK_TILE              = 0x0400
	GC_MASK_STIPPLE           = 0x0800
	GC_MASK_TILE_STIP_XORIGIN = 0x1000
	GC_MASK_TILE_STIP_YORIGIN = 0x2000
	GC_MASK_FONT              = 0x4000
)

type CreateGCRequest struct {
	// fixed struct part
	Gcid      base.GC
	Drawable  base.DRAWABLE
	ValueMask base.CARD32

	// optional data
	Function        base.CARD32
	PlaneMask       base.CARD32
	Foreground      base.CARD32
	Background      base.CARD32
	LineWidth       base.CARD32
	LineStyle       base.CARD32
	CapStyle        base.CARD32
	JoinStyle       base.CARD32
	FillStyle       base.CARD32
	FillRule        base.CARD32
	Tile            base.CARD32
	Stipple         base.CARD32
	TileStipXOrigin base.CARD32
	TileStipYOrigin base.CARD32
	Font            base.FONT
}

func (r CreateGCRequest) IsMask(m base.CARD32) bool {
	return (r.ValueMask & m) == m
}

func (r *CreateGCRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.CreateGC)
	writer.WriteGC(r.Gcid)
	writer.WriteDRAWABLE(r.Drawable)
	writer.WriteCARD32(r.ValueMask)

	if r.IsMask(GC_MASK_FUNCTION) {
		writer.WriteCARD32(r.Function)
	}
	if r.IsMask(GC_MASK_PLANE_MASK) {
		writer.WriteCARD32(r.PlaneMask)
	}
	if r.IsMask(GC_MASK_FOREGROUND) {
		writer.WriteCARD32(r.Foreground)
	}
	if r.IsMask(GC_MASK_BACKGROUND) {
		writer.WriteCARD32(r.Background)
	}
	if r.IsMask(GC_MASK_LINE_WIDTH) {
		writer.WriteCARD32(r.LineWidth)
	}
	if r.IsMask(GC_MASK_LINE_STYLE) {
		writer.WriteCARD32(r.LineStyle)
	}
	if r.IsMask(GC_MASK_CAP_STYLE) {
		writer.WriteCARD32(r.CapStyle)
	}
	if r.IsMask(GC_MASK_JOIN_STYLE) {
		writer.WriteCARD32(r.JoinStyle)
	}
	if r.IsMask(GC_MASK_FILL_STYLE) {
		writer.WriteCARD32(r.FillStyle)
	}
	if r.IsMask(GC_MASK_FILL_RULE) {
		writer.WriteCARD32(r.FillRule)
	}
	if r.IsMask(GC_MASK_TILE) {
		writer.WriteCARD32(r.Tile)
	}
	if r.IsMask(GC_MASK_STIPPLE) {
		writer.WriteCARD32(r.Stipple)
	}
	if r.IsMask(GC_MASK_TILE_STIP_XORIGIN) {
		writer.WriteCARD32(r.TileStipXOrigin)
	}
	if r.IsMask(GC_MASK_TILE_STIP_YORIGIN) {
		writer.WriteCARD32(r.TileStipYOrigin)
	}
	if r.IsMask(GC_MASK_FONT) {
		writer.WriteCARD32(base.CARD32(r.Font))
	}

	return nil
}
