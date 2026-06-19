package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetSelectionOwnerRequest struct {
	Selection base.ATOM
}

func (r *GetSelectionOwnerRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetSelectionOwner)
	writer.WriteATOM(r.Selection)
	return nil
}

type GetSelectionOwnerReply struct {
	Owner base.WINDOW
}

func (reply *GetSelectionOwnerReply) Parse(reader base.ReplyReader) error {
	reply.Owner = reader.XID()
	return reader.LastError
}
