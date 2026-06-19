package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// Blanking / exposures preference (No=0, Yes=1, Default=2).
const (
	BlankingNo      = 0
	BlankingYes     = 1
	BlankingDefault = 2
)

type SetScreenSaverRequest struct {
	Timeout        base.INT16
	Interval       base.INT16
	PreferBlanking base.CARD8
	AllowExposures base.CARD8
}

func (r *SetScreenSaverRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.SetScreenSaver)
	writer.WriteINT16(r.Timeout)
	writer.WriteINT16(r.Interval)
	writer.WriteCARD8(r.PreferBlanking)
	writer.WriteCARD8(r.AllowExposures)
	writer.WriteCARD16(0) // unused
	return nil
}
