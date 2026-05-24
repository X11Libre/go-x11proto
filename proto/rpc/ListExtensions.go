package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ListExtensions(c *core.X11Conn) ([]string, error) {
	req := request.ListExtensionsRequest{}

	reply, err := c.SendAndWait(&req)
	if err != nil {
		return []string{}, err
	}

	rep := request.ListExtensionsReply{}
	if err := rep.Parse(*reply); err != nil {
		return []string{}, err
	}

	return rep.Names, nil
}
