package setup

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type XSetupScreen struct {
	RootWindow   base.WINDOW
	Colormap     base.COLORMAP
	WhitePixel   base.CARD32
	BlackPixel   base.CARD32
	InputMasks   base.CARD32
	Width        base.CARD16
	Height       base.CARD16
	WidthMM      base.CARD16
	HeightMM     base.CARD16
	MinColormaps base.CARD16
	MaxColormaps base.CARD16
	RootVisual   base.VISUAL
	BackingStore base.CARD8
	SaveUnder    base.CARD8
	RootDepth    base.CARD8
	NumDepths    base.CARD8

	Depths []XSetupScreenDepth
}

func (s *XSetupScreen) Parse(readbuf *base.ReadBuffer) error {
	s.RootWindow = readbuf.WINDOW()
	s.Colormap = readbuf.COLORMAP()
	s.WhitePixel = readbuf.CARD32()
	s.BlackPixel = readbuf.CARD32()
	s.InputMasks = readbuf.CARD32()
	s.Width = readbuf.CARD16()
	s.Height = readbuf.CARD16()
	s.WidthMM = readbuf.CARD16()
	s.HeightMM = readbuf.CARD16()
	s.MinColormaps = readbuf.CARD16()
	s.MaxColormaps = readbuf.CARD16()
	s.RootVisual = readbuf.VISUAL()
	s.BackingStore = readbuf.CARD8()
	s.SaveUnder = readbuf.CARD8()
	s.RootDepth = readbuf.CARD8()
	s.NumDepths = readbuf.CARD8()

	for i := base.CARD8(0); i < s.NumDepths; i++ {
		d := XSetupScreenDepth{}
		d.Parse(readbuf)
		s.Depths = append(s.Depths, d)
	}
	return readbuf.LastError
}
