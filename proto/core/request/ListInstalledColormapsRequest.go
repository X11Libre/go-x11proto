package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ListInstalledColormapsRequest struct {
	Window base.WINDOW
}

func (r *ListInstalledColormapsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ListInstalledMaps)
	writer.WriteXID(r.Window)
	return nil
}

type ListInstalledColormapsReply struct {
	Colormaps []base.COLORMAP
}

func (reply *ListInstalledColormapsReply) Parse(reader base.ReplyReader) error {
	n := int(reader.CARD16())
	reader.ReadBytes(22) // unused
	reply.Colormaps = make([]base.COLORMAP, 0, n)
	for i := 0; i < n; i++ {
		reply.Colormaps = append(reply.Colormaps, base.COLORMAP(reader.CARD32()))
	}
	return reader.LastError
}
