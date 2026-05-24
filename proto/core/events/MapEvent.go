package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type MapEvent struct {
	GenericEvent
	Serial           base.CARD32
	EventWindow      base.WINDOW
	MappedWindow     base.WINDOW
	OverrideRedirect bool
}

func (e *MapEvent) ReceiverWindow() base.WINDOW {
	return e.EventWindow
}

func (e *MapEvent) Parse(rbuf base.ReadBuffer) error {
	e.Serial = rbuf.CARD32()
	e.EventWindow = rbuf.WINDOW()
	e.MappedWindow = rbuf.WINDOW()
	e.OverrideRedirect = rbuf.Bool()
	return rbuf.LastError
}

func ParseEvent_MapNotify(gev GenericEvent, rbuf base.ReadBuffer) (*MapEvent, error) {
	ev := MapEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}
