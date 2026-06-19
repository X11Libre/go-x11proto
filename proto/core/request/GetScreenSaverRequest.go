package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetScreenSaverRequest struct{}

func (r *GetScreenSaverRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetScreenSaver)
	return nil
}

type GetScreenSaverReply struct {
	Timeout        base.CARD16
	Interval       base.CARD16
	PreferBlanking base.CARD8
	AllowExposures base.CARD8
}

func (reply *GetScreenSaverReply) Parse(reader base.ReplyReader) error {
	reply.Timeout = reader.CARD16()
	reply.Interval = reader.CARD16()
	reply.PreferBlanking = reader.CARD8()
	reply.AllowExposures = reader.CARD8()
	return reader.LastError
}
