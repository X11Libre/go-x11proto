package setup

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type XSetupPrefix struct {
	Status       base.CARD8
	ReasonLength base.CARD8
	MajorVersion base.CARD16
	MinorVersion base.CARD16
	Length       base.CARD16
}

func (s *XSetupPrefix) ParseBytes(data []byte, be bool) error {
	readbuf := base.MakeReadBuffer(data, be)
	s.Status = readbuf.CARD8()
	s.ReasonLength = readbuf.CARD8()
	s.MajorVersion = readbuf.CARD16()
	s.MinorVersion = readbuf.CARD16()
	s.Length = readbuf.CARD16()
	return readbuf.LastError
}
