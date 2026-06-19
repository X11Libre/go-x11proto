package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetSelectionOwner(c *core.X11Conn, selection base.ATOM) (base.WINDOW, error) {
	reply, err := c.SendAndWait(&request.GetSelectionOwnerRequest{Selection: selection})
	if err != nil {
		return 0, err
	}
	rep := &request.GetSelectionOwnerReply{}
	if err := rep.Parse(*reply); err != nil {
		return 0, err
	}
	return rep.Owner, nil
}
