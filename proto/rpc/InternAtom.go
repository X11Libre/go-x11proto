package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func InternAtom(c *core.X11Conn, name string) (base.ATOM, error) {
	req := request.InternAtomRequest{
		Name: name,
	}

	reply, err := c.SendAndWait(&req)
	if err != nil {
		return 0, err
	}

	rep := request.InternAtomReply{}
	if err := rep.Parse(*reply); err != nil {
		return 0, err
	}

	return rep.Atom, nil
}

func GetAtom(c *core.X11Conn, name string) (base.ATOM, error) {
	if val, ok := c.AtomCache[name]; ok {
		return val, nil
	}

	val, err := InternAtom(c, name)
	if err != nil {
		return 0, err
	}

	c.AtomCache[name] = val
	return val, nil
}
