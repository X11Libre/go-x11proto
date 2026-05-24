package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type ConfigureEvent struct {
	GenericEvent
	EventWindow      base.WINDOW
	TargetWindow     base.WINDOW
	AboveSibling     base.WINDOW
	X                base.INT16
	Y                base.INT16
	Width            base.CARD16
	Height           base.CARD16
	BorderWidth      base.CARD16
	OverrideRedirect bool
}

func (e *ConfigureEvent) ReceiverWindow() base.WINDOW {
	return e.TargetWindow
}

func (e *ConfigureEvent) Parse(rbuf base.ReadBuffer) error {
	e.EventWindow = rbuf.WINDOW()
	e.TargetWindow = rbuf.WINDOW()
	e.AboveSibling = rbuf.WINDOW()
	e.X = rbuf.INT16()
	e.Y = rbuf.INT16()
	e.Width = rbuf.CARD16()
	e.Height = rbuf.CARD16()
	e.BorderWidth = rbuf.CARD16()
	e.OverrideRedirect = rbuf.Bool()
	return rbuf.LastError
}

func ParseEvent_ConfigureNotify(gev GenericEvent, rbuf base.ReadBuffer) (*ConfigureEvent, error) {
	ev := ConfigureEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}
