package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetMotionEventsRequest struct {
	Window base.WINDOW
	Start  base.CARD32
	Stop   base.CARD32
}

func (r *GetMotionEventsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetMotionEvents)
	writer.WriteXID(r.Window)
	writer.WriteCARD32(r.Start)
	writer.WriteCARD32(r.Stop)
	return nil
}

// TimeCoord is a single timestamped pointer position (TIMECOORD).
type TimeCoord struct {
	Time base.CARD32
	X    base.INT16
	Y    base.INT16
}

type GetMotionEventsReply struct {
	Events []TimeCoord
}

func (reply *GetMotionEventsReply) Parse(reader base.ReplyReader) error {
	n := int(reader.CARD32())
	reader.ReadBytes(20) // unused
	reply.Events = make([]TimeCoord, 0, n)
	for i := 0; i < n; i++ {
		reply.Events = append(reply.Events, TimeCoord{
			Time: reader.CARD32(),
			X:    reader.INT16(),
			Y:    reader.INT16(),
		})
	}
	return reader.LastError
}
