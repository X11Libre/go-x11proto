package setup

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type XSetupScreenDepth struct {
	Depth      base.CARD8
	Pad0       base.CARD8
	NumVisuals base.CARD16
	Pad1       base.CARD32

	Visuals []XSetupScreenVisual
}

func (d *XSetupScreenDepth) Parse(readbuf *base.ReadBuffer) error {
	d.Depth = readbuf.CARD8()
	d.Pad0 = readbuf.CARD8() // padding
	d.NumVisuals = readbuf.CARD16()
	d.Pad1 = readbuf.CARD32() // padding

	for i := base.CARD16(0); i < d.NumVisuals; i++ {
		v := XSetupScreenVisual{}
		v.Parse(readbuf)
		d.Visuals = append(d.Visuals, v)
	}

	return readbuf.LastError
}
