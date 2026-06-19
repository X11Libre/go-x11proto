package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// ChangeHosts mode (param0).
const (
	HostInsert = 0
	HostDelete = 1
)

// Host families.
const (
	FamilyInternet          = 0
	FamilyDECnet            = 1
	FamilyChaos             = 2
	FamilyServerInterpreted = 5
	FamilyInternet6         = 6
)

type ChangeHostsRequest struct {
	Mode    base.CARD8
	Family  base.CARD8
	Address []byte
}

func (r *ChangeHostsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ChangeHosts)
	writer.SetParam0(r.Mode)
	writer.WriteCARD8(r.Family)
	writer.WriteCARD8(0) // unused
	writer.WriteCARD16(base.CARD16(len(r.Address)))
	writer.WriteBytes(r.Address)
	writer.Pad()
	return nil
}
