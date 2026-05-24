package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/atoms"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ChangeProperty8(c *core.X11Conn, mode base.CARD8, window base.WINDOW, name base.ATOM, ptype base.ATOM, data []base.CARD8) error {
	req := request.ChangePropertyRequest{
		Mode:     mode,
		Format:   8,
		Window:   window,
		Property: name,
		Type:     ptype,
		Data8:    data,
	}

	_, err := c.Send(&req)
	return err
}

func ChangeProperty16(c *core.X11Conn, mode base.CARD8, window base.WINDOW, name base.ATOM, ptype base.ATOM, data []base.CARD16) error {
	req := request.ChangePropertyRequest{
		Mode:     mode,
		Format:   16,
		Window:   window,
		Property: name,
		Type:     ptype,
		Data16:   data,
	}

	_, err := c.Send(&req)
	return err
}

func ChangeProperty32(c *core.X11Conn, mode base.CARD8, window base.WINDOW, name base.ATOM, ptype base.ATOM, data []base.CARD32) error {
	req := request.ChangePropertyRequest{
		Mode:     mode,
		Format:   32,
		Window:   window,
		Property: name,
		Type:     ptype,
		Data32:   data,
	}

	_, err := c.Send(&req)
	return err
}

func ChangePropertyString(c *core.X11Conn, mode base.CARD8, window base.WINDOW, name base.ATOM, str string) error {
	return ChangeProperty8(c, mode, window, name, atoms.STRING, []base.CARD8(str))
}
