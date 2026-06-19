package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GrabKeyboard(c *core.X11Conn, ownerEvents bool, grabWindow base.WINDOW, time base.CARD32, pointerMode, keyboardMode base.CARD8) (*request.GrabKeyboardReply, error) {
	reply, err := c.SendAndWait(&request.GrabKeyboardRequest{
		OwnerEvents: ownerEvents, GrabWindow: grabWindow, Time: time,
		PointerMode: pointerMode, KeyboardMode: keyboardMode,
	})
	if err != nil {
		return nil, err
	}
	rep := &request.GrabKeyboardReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
