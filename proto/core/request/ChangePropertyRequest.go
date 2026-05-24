package request

import (
	"fmt"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

const (
	CHANGE_PROPERTY_REPLACE = 0
	CHANGE_PROPERTY_PREPEND = 1
	CHANGE_PROPERTY_APPEND  = 2
)

type ChangePropertyRequest struct {
	Mode     base.CARD8
	Window   base.WINDOW
	Property base.ATOM
	Type     base.ATOM
	Format   base.CARD8

	Data8  []base.CARD8
	Data16 []base.CARD16
	Data32 []base.CARD32
}

func (r *ChangePropertyRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ChangeProperty)
	writer.SetParam0(r.Mode)
	writer.WriteWINDOW(r.Window)
	writer.WriteATOM(r.Property)
	writer.WriteATOM(r.Type)
	writer.WriteCARD8(r.Format)
	writer.Pad()

	switch r.Format {
	case 8:
		writer.WriteCARD32(base.CARD32(len(r.Data8)))
	case 16:
		writer.WriteCARD32(base.CARD32(len(r.Data16)))
	case 32:
		writer.WriteCARD32(base.CARD32(len(r.Data32)))
	default:
		return fmt.Errorf("unsupported format %d", r.Format)
	}

	writer.Pad()

	switch r.Format {
	case 8:
		writer.WriteCARD8s(r.Data8)
	case 16:
		writer.WriteCARD16s(r.Data16)
	case 32:
		writer.WriteCARD32s(r.Data32)
	}

	writer.Pad()
	return nil
}
