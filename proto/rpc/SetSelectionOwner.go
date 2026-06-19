package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func SetSelectionOwner(c *core.X11Conn, owner base.WINDOW, selection base.ATOM, time base.CARD32) error {
	_, err := c.Send(&request.SetSelectionOwnerRequest{Owner: owner, Selection: selection, Time: time})
	return err
}
