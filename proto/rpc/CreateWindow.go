package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func CreateWindow1(c *core.X11Conn, parent base.WINDOW, x, y int16, width, height uint16, evMask base.CARD32) (base.WINDOW, error) {
	winId := c.NextResourceID()
	req := request.CreateWindowRequest{
		Depth:     0,
		Wid:       base.WINDOW(winId),
		Parent:    parent,
		X:         base.INT16(x),
		Y:         base.INT16(y),
		Width:     base.CARD16(width),
		Height:    base.CARD16(height),
		Border:    1,
		Class:     request.WindowClass_InputOutput,
		Visual:    0,
		ValueMask: request.CW_BACKGROUND_PIXEL | request.CW_EVENT_MASK,
		BackPixel: c.Setup.Screens[0].WhitePixel,
		EventMask: evMask,
	}

	_, err := c.Send(&req)
	return req.Wid, err
}
