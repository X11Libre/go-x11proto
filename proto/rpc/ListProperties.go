package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ListProperties(c *core.X11Conn, win base.WINDOW) ([]base.ATOM, error) {
	reply, err := c.SendAndWait(&request.ListPropertiesRequest{Window: win})
	if err != nil {
		return nil, err
	}
	rep := &request.ListPropertiesReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep.Atoms, nil
}
