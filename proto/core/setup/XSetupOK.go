package setup

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type XSetupOK struct {
	/* these are directly read as is from buffer */
	Release          base.CARD32
	RidBase          base.CARD32
	RidMask          base.CARD32
	MotionBufferSize base.CARD32

	VendorLength   base.CARD16
	MaxRequestSize base.CARD16

	NumRoots   base.CARD8
	NumFormats base.CARD8

	ImageByteOrder base.CARD8
	BitmapBitOrder base.CARD8

	BitmapScanlineUnit base.CARD8
	BitmapScanlinePad  base.CARD8

	MinKeycode base.CARD8
	MaxKeycode base.CARD8

	/* these are explicitly parsed */
	VendorName string

	PixmapFormats []XSetupPixmapFormat
	Screens       []XSetupScreen
}

func (s *XSetupOK) ParseBytes(data []byte, be bool) error {
	readbuf := base.MakeReadBuffer(data, be)
	return s.Parse(&readbuf)
}

func (s *XSetupOK) Parse(readbuf *base.ReadBuffer) error {
	s.Release = readbuf.CARD32()
	s.RidBase = readbuf.CARD32()
	s.RidMask = readbuf.CARD32()
	s.MotionBufferSize = readbuf.CARD32()
	s.VendorLength = readbuf.CARD16()
	s.MaxRequestSize = readbuf.CARD16()
	s.NumRoots = readbuf.CARD8()
	s.NumFormats = readbuf.CARD8()
	s.ImageByteOrder = readbuf.CARD8()
	s.BitmapBitOrder = readbuf.CARD8()
	s.BitmapScanlineUnit = readbuf.CARD8()
	s.BitmapScanlinePad = readbuf.CARD8()
	s.MinKeycode = readbuf.CARD8()
	s.MaxKeycode = readbuf.CARD8()
	readbuf.CARD32() /* pad0 */

	padsz := base.RoundFullUnits(uint(s.VendorLength))
	s.VendorName = string(readbuf.ReadBytes(padsz))

	// parse the pixmap formats
	for idx := base.CARD8(0); idx < s.NumFormats; idx++ {
		pixfmt := XSetupPixmapFormat{}
		pixfmt.Parse(readbuf)
		s.PixmapFormats = append(s.PixmapFormats, pixfmt)
	}

	// parse the screen structures
	for idx := base.CARD8(0); idx < s.NumRoots; idx++ {
		scrn := XSetupScreen{}
		scrn.Parse(readbuf)
		s.Screens = append(s.Screens, scrn)
	}
	return readbuf.LastError
}
