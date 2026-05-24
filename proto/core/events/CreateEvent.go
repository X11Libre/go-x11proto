package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type CreateEvent struct {
	GenericEvent
	ParentWindow base.WINDOW
	TargetWindow base.WINDOW
	X base.INT16
	Y base.INT16
	Width base.CARD16
	Height base.CARD16
	BorderWidth base.CARD16
	OverrideRedirect bool

}

func (e *CreateEvent) ReceiverWindow() base.WINDOW {
	return e.TargetWindow
}

func (e *CreateEvent) Parse(rbuf base.ReadBuffer) error {
	e.ParentWindow = rbuf.WINDOW()
	e.TargetWindow = rbuf.WINDOW()
	e.X = rbuf.INT16()
	e.Y = rbuf.INT16()
	e.Width = rbuf.CARD16()
	e.Height = rbuf.CARD16()
	e.BorderWidth = rbuf.CARD16()
	e.OverrideRedirect = rbuf.Bool()
	return rbuf.LastError
}

func ParseEvent_CreateNotify(gev GenericEvent, rbuf base.ReadBuffer) (*CreateEvent, error) {
	ev := CreateEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}
