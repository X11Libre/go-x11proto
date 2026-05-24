package core

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
)

type X11WindowEventHandler interface {
	HandleX11WindowEvent(window base.WINDOW, ev events.Event) bool
}
