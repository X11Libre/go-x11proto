package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ListFonts(c *core.X11Conn, maxNames base.CARD16, pattern string) ([]string, error) {
	reply, err := c.SendAndWait(&request.ListFontsRequest{MaxNames: maxNames, Pattern: pattern})
	if err != nil {
		return nil, err
	}
	rep := &request.ListFontsReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep.Names, nil
}
