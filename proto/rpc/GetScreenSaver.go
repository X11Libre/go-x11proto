package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetScreenSaver(c *core.X11Conn) (*request.GetScreenSaverReply, error) {
	reply, err := c.SendAndWait(&request.GetScreenSaverRequest{})
	if err != nil {
		return nil, err
	}
	rep := &request.GetScreenSaverReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
