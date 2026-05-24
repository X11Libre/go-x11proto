package setup

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type XSetupPixmapFormat struct {
	Depth        base.CARD8
	BitsPerPixel base.CARD8
	ScanlinePad  base.CARD8
}

func (pf *XSetupPixmapFormat) Parse(readbuf *base.ReadBuffer) error {
	pf.Depth = readbuf.CARD8()
	pf.BitsPerPixel = readbuf.CARD8()
	pf.ScanlinePad = readbuf.CARD8()

	readbuf.CARD8()  /* pad0 */
	readbuf.CARD32() /* pad1 */
	return readbuf.LastError
}
