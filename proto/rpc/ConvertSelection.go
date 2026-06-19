package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ConvertSelection(c *core.X11Conn, requestor base.WINDOW, selection, target, property base.ATOM, time base.CARD32) error {
	_, err := c.Send(&request.ConvertSelectionRequest{
		Requestor: requestor, Selection: selection, Target: target, Property: property, Time: time,
	})
	return err
}
