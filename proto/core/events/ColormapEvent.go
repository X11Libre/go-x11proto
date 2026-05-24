package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type ColormapEvent struct {
	GenericEvent
	Window    base.WINDOW
	Colormap  base.COLORMAP
	New       bool
	Installed bool
}

func (e *ColormapEvent) ReceiverWindow() base.WINDOW {
	return e.Window
}

func (e *ColormapEvent) Parse(rbuf base.ReadBuffer) error {
	e.Window = rbuf.WINDOW()
	e.Colormap = rbuf.COLORMAP()
	e.New = rbuf.Bool()
	e.Installed = rbuf.Bool()
	return rbuf.LastError
}

func ParseEvent_ColormapNotify(gev GenericEvent, rbuf base.ReadBuffer) (*ColormapEvent, error) {
	ev := ColormapEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}
