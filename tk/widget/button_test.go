package widget

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
)

// newButton builds a Button with bounds but no server resources (gc==nil, so
// Repaint is a no-op), to drive the press/drag/release logic offline.
func newButton() (*Button, *int) {
	b := &Button{}
	b.W, b.H = 80, 30
	fired := new(int)
	b.OnButtonPress = func() { *fired++ }
	return b, fired
}

func press(x, y base.CARD16) *events.ButtonPressEvent {
	e := &events.ButtonPressEvent{}
	e.Key, e.EventX, e.EventY = 1, x, y
	return e
}
func release(x, y base.CARD16) *events.ButtonReleaseEvent {
	e := &events.ButtonReleaseEvent{}
	e.Key, e.EventX, e.EventY = 1, x, y
	return e
}
func motion(x, y base.CARD16) *events.MotionEvent {
	e := &events.MotionEvent{}
	e.EventX, e.EventY = x, y
	return e
}

func TestButtonClickInside(t *testing.T) {
	b, fired := newButton()
	b.HandleWindowEvent(press(10, 10))
	b.HandleWindowEvent(release(20, 15))
	if *fired != 1 {
		t.Errorf("press+release inside should fire once, got %d", *fired)
	}
}

func TestButtonReleaseOutsideCancels(t *testing.T) {
	b, fired := newButton()
	b.HandleWindowEvent(press(10, 10))
	b.HandleWindowEvent(motion(200, 10)) // drag out
	if b.down {
		t.Error("button should pop up when the pointer leaves")
	}
	b.HandleWindowEvent(release(200, 10)) // release outside
	if *fired != 0 {
		t.Errorf("release outside must not fire, got %d", *fired)
	}
}

func TestButtonDragOutAndBackInFires(t *testing.T) {
	b, fired := newButton()
	b.HandleWindowEvent(press(10, 10))
	b.HandleWindowEvent(motion(200, 10)) // out
	if b.down {
		t.Error("should be up while outside")
	}
	b.HandleWindowEvent(motion(15, 12)) // back in
	if !b.down {
		t.Error("should be down again when back inside")
	}
	b.HandleWindowEvent(release(15, 12))
	if *fired != 1 {
		t.Errorf("release inside after drag-back should fire, got %d", *fired)
	}
}

func TestButtonReleaseWithoutPressIgnored(t *testing.T) {
	b, fired := newButton()
	b.HandleWindowEvent(release(10, 10)) // stray release, no prior press
	if *fired != 0 {
		t.Errorf("release without press must not fire, got %d", *fired)
	}
}
