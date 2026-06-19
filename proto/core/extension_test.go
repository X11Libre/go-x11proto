package core

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
)

func TestExtensionEventRouting(t *testing.T) {
	c := &X11Conn{}
	called := false
	// extension occupies event codes [85, 88]
	c.RegisterExtensionEvents(85, 4, func(data []byte, be bool) (events.Event, error) {
		called = true
		return events.GenericEvent{}, nil
	})

	cases := map[base.CARD8]bool{84: false, 85: true, 88: true, 89: false}
	for code, want := range cases {
		if got := c.extEventParser(code) != nil; got != want {
			t.Errorf("extEventParser(%d): in-range=%v, want %v", code, got, want)
		}
	}

	if _, err := c.extEventParser(86)(make([]byte, 32), false); err != nil {
		t.Fatalf("parser error: %v", err)
	}
	if !called {
		t.Error("registered parser was not invoked")
	}
}

func TestExtensionErrorRouting(t *testing.T) {
	c := &X11Conn{}
	c.RegisterExtensionErrors(150, 3, "XYZ") // codes [150, 152]
	cases := map[base.CARD8]string{149: "", 150: "XYZ", 152: "XYZ", 153: ""}
	for code, want := range cases {
		if got := c.extErrorName(code); got != want {
			t.Errorf("extErrorName(%d) = %q, want %q", code, got, want)
		}
	}
}

func TestRegisterExtensionEventsGuards(t *testing.T) {
	c := &X11Conn{}
	c.RegisterExtensionEvents(10, 0, func([]byte, bool) (events.Event, error) { return nil, nil }) // count 0: ignored
	c.RegisterExtensionEvents(10, 2, nil)                                                          // nil parser: ignored
	if c.extEventParser(10) != nil {
		t.Error("no parser should have been registered")
	}
}
