package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/atoms"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func SetWindowName(conn *core.X11Conn, window base.WINDOW, name string) error {
	if err := ChangePropertyString(
		conn,
		request.CHANGE_PROPERTY_REPLACE,
		window,
		atoms.WM_NAME,
		name); err != nil {
		return err
	}

	netwm_name, err := GetAtom(conn, "_NET_WM_NAME")
	if err != nil {
		return err
	}

	if err = ChangePropertyString(
		conn,
		request.CHANGE_PROPERTY_REPLACE,
		window,
		netwm_name,
		name); err != nil {
		return err
	}

	return nil
}
