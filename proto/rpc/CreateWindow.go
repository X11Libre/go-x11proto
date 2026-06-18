package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

// WindowSpec describes an InputOutput child window for CreateWindow. The event
// mask is always selected (CW_EVENT_MASK); the background and border pixels are
// applied only when their Set flag is set, so a black (0) pixel can be
// distinguished from "leave unset".
type WindowSpec struct {
	Parent         base.WINDOW
	X, Y           int16
	Width, Height  uint16
	BorderWidth    uint16
	EventMask      base.CARD32
	BackPixel      base.CARD32
	SetBackPixel   bool
	BorderPixel    base.CARD32
	SetBorderPixel bool
}

// CreateWindow allocates an XID and creates the window described by spec,
// returning the new window id. It fills the defaults the demos need (depth 0,
// InputOutput, copy-from-parent visual) and builds the value mask from the
// fields that are set.
func CreateWindow(c *core.X11Conn, spec WindowSpec) (base.WINDOW, error) {
	req := request.CreateWindowRequest{
		Depth:     0,
		Wid:       base.WINDOW(c.NextResourceID()),
		Parent:    spec.Parent,
		X:         base.INT16(spec.X),
		Y:         base.INT16(spec.Y),
		Width:     base.CARD16(spec.Width),
		Height:    base.CARD16(spec.Height),
		Border:    base.CARD16(spec.BorderWidth),
		Class:     request.WindowClass_InputOutput,
		Visual:    0,
		ValueMask: request.CW_EVENT_MASK,
		EventMask: spec.EventMask,
	}
	if spec.SetBackPixel {
		req.ValueMask |= request.CW_BACKGROUND_PIXEL
		req.BackPixel = spec.BackPixel
	}
	if spec.SetBorderPixel {
		req.ValueMask |= request.CW_BORDER_PIXEL
		req.BorderPixel = spec.BorderPixel
	}
	_, err := c.Send(&req)
	return req.Wid, err
}

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
