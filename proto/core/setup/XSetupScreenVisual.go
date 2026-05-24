package setup

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type XSetupScreenVisual struct {
	Id          base.VISUAL
	Class       base.CARD8
	BitsPerChan base.CARD8
	NumColors   base.CARD16
	RedMask     base.CARD32
	GreenMask   base.CARD32
	BlueMask    base.CARD32
	Padding     base.CARD32
}

func (v *XSetupScreenVisual) Parse(readbuf *base.ReadBuffer) error {
	v.Id = readbuf.VISUAL()
	v.Class = readbuf.CARD8()
	v.BitsPerChan = readbuf.CARD8()
	v.NumColors = readbuf.CARD16()
	v.RedMask = readbuf.CARD32()
	v.GreenMask = readbuf.CARD32()
	v.BlueMask = readbuf.CARD32()
	v.Padding = readbuf.CARD32()
	return readbuf.LastError
}
