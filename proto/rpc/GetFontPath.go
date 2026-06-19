package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetFontPath(c *core.X11Conn) ([]string, error) {
	reply, err := c.SendAndWait(&request.GetFontPathRequest{})
	if err != nil {
		return nil, err
	}
	rep := &request.GetFontPathReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep.Path, nil
}
