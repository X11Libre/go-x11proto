package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ListHostsRequest struct{}

func (r *ListHostsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ListHosts)
	return nil
}

// Host is an entry in the access control list.
type Host struct {
	Family  base.CARD8
	Address []byte
}

type ListHostsReply struct {
	Enabled bool
	Hosts   []Host
}

func (reply *ListHostsReply) Parse(reader base.ReplyReader) error {
	reply.Enabled = reader.Data0 != 0
	n := int(reader.CARD16())
	reader.ReadBytes(22) // unused
	reply.Hosts = make([]Host, 0, n)
	for i := 0; i < n; i++ {
		family := reader.CARD8()
		reader.ReadBytes(1) // unused
		l := uint(reader.CARD16())
		addr := reader.ReadBytes(l)
		// each HOST is padded to a 4-byte boundary
		if pad := (4 - (l % 4)) % 4; pad != 0 {
			reader.ReadBytes(pad)
		}
		reply.Hosts = append(reply.Hosts, Host{Family: family, Address: addr})
	}
	return reader.LastError
}
