package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetWindowAttributesRequest struct {
	Window base.WINDOW
}

func (r *GetWindowAttributesRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetWindowAttr)
	writer.WriteXID(r.Window)
	return nil
}

type GetWindowAttributesReply struct {
	BackingStore       base.CARD8 // reply header byte
	Visual             base.VISUAL
	Class              base.CARD16
	BitGravity         base.CARD8
	WinGravity         base.CARD8
	BackingPlanes      base.CARD32
	BackingPixel       base.CARD32
	SaveUnder          bool
	MapIsInstalled     bool
	MapState           base.CARD8
	OverrideRedirect   bool
	Colormap           base.COLORMAP
	AllEventMasks      base.CARD32
	YourEventMask      base.CARD32
	DoNotPropagateMask base.CARD16
}

func (reply *GetWindowAttributesReply) Parse(reader base.ReplyReader) error {
	reply.BackingStore = reader.Data0
	reply.Visual = base.VISUAL(reader.CARD32())
	reply.Class = reader.CARD16()
	reply.BitGravity = reader.CARD8()
	reply.WinGravity = reader.CARD8()
	reply.BackingPlanes = reader.CARD32()
	reply.BackingPixel = reader.CARD32()
	reply.SaveUnder = reader.Bool()
	reply.MapIsInstalled = reader.Bool()
	reply.MapState = reader.CARD8()
	reply.OverrideRedirect = reader.Bool()
	reply.Colormap = base.COLORMAP(reader.CARD32())
	reply.AllEventMasks = reader.CARD32()
	reply.YourEventMask = reader.CARD32()
	reply.DoNotPropagateMask = reader.CARD16()
	return reader.LastError
}
