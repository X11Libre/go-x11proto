package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetAtomName(c *core.X11Conn, atom base.ATOM) (string, error) {
	reply, err := c.SendAndWait(&request.GetAtomNameRequest{Atom: atom})
	if err != nil {
		return "", err
	}
	rep := &request.GetAtomNameReply{}
	if err := rep.Parse(*reply); err != nil {
		return "", err
	}
	return rep.Name, nil
}
