package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type ExposeEvent struct {
	GenericEvent
	Window base.WINDOW
	X, Y   base.CARD16
	Width  base.CARD16
	Height base.CARD16
	Count  base.CARD16
}

func (ev *ExposeEvent) ReceiverWindow() base.WINDOW {
	return ev.Window
}

func (ev *ExposeEvent) Parse(rbuf base.ReadBuffer) error {
	ev.Window = rbuf.WINDOW()
	ev.X = rbuf.CARD16()
	ev.Y = rbuf.CARD16()
	ev.Width = rbuf.CARD16()
	ev.Height = rbuf.CARD16()
	ev.Count = rbuf.CARD16()
	return rbuf.LastError
}

func ParseEvent_Expose(gev GenericEvent, rbuf base.ReadBuffer) (*ExposeEvent, error) {
	ev := ExposeEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}
