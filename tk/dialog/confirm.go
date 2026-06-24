package dialog

import (
	"strings"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font"
	"github.com/X11Libre/go-x11proto/tk/keyboard"
)

// Confirm is a small modal yes/no dialog: a message (one or more lines) and a
// key hint. Enter or 'y' confirms (OnYes); Escape or 'n' cancels (OnNo). Like
// FilePicker it draws itself and does not self-destroy — the owner closes it
// from the callbacks.
//
// Fill the embedded Window (Parent/X/Y/W/H) and Font before Init. Set Floating
// (with an optional Title) for a separate, window-manager-managed window.
type Confirm struct {
	tk_core.Window
	Font     *font.Font
	Keymap   *keyboard.Map
	Message  string
	OnYes    func()
	OnNo     func()
	Floating bool
	Title    string

	gc *tk_core.GC
}

// Init creates and maps the dialog and takes focus.
func (c *Confirm) Init() error {
	c.EventMask |= base.CARD32(event_mask.Exposure | event_mask.KeyPress)
	c.SetBackPixel = true
	c.BackPixel = c.Conn.X11Conn.DefaultWhitePixel()
	c.SetBorderPixel = true
	c.BorderPixel = c.Conn.X11Conn.DefaultBlackPixel()
	c.BorderWidth = 1

	if c.Floating {
		c.Parent = nil
		c.ParentXID = 0
		if c.Title == "" {
			c.Title = "Confirm"
		}
		c.Name = c.Title
	}

	c.SetWindowHandler(c)
	if err := c.Window.Create(); err != nil {
		return err
	}
	gc, err := c.Conn.CreateGC1(c.Conn.X11Conn.DefaultBlackPixel(),
		c.Conn.X11Conn.DefaultWhitePixel(), c.Font.ID)
	if err != nil {
		return err
	}
	c.gc = gc
	if c.Keymap == nil {
		if km, err := keyboard.Load(c.Conn.X11Conn); err == nil {
			c.Keymap = km
		}
	}
	if err := c.Window.Map(); err != nil {
		return err
	}
	c.Focus()
	return c.Draw()
}

// Focus requests the keyboard focus.
func (c *Confirm) Focus() {
	_ = rpc.SetInputFocus(c.Conn.X11Conn, 2 /*RevertToParent*/, c.XID, 0)
}

// Draw paints the message lines and the key hint.
func (c *Confirm) Draw() error {
	if c.gc == nil {
		return nil
	}
	if err := c.ClearArea(0, 0, 0, 0, false); err != nil {
		return err
	}
	lh := c.Font.Height()
	asc := c.Font.Ascent
	y := 6 + asc
	for _, line := range strings.Split(c.Message, "\n") {
		if err := c.PutText8(c.gc.XID, 8, base.INT16(y), line); err != nil {
			return err
		}
		y += lh
	}
	hintY := int(c.H) - lh
	c.FillRect(c.gc.XID, 0, base.INT16(hintY-2), c.W, 1)
	return c.PutText8(c.gc.XID, 8, base.INT16(hintY+asc), "Enter / y: yes      Esc / n: no")
}

// HandleWindowEvent confirms on Enter/'y', cancels on Escape/'n'.
func (c *Confirm) HandleWindowEvent(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		_ = c.Draw()
	case *events.KeyPressEvent:
		if c.Keymap == nil {
			return true
		}
		k := c.Keymap.Lookup(e.Key, e.State)
		switch {
		case k.Key == keyboard.KeyEnter, k.Rune == 'y', k.Rune == 'Y':
			if c.OnYes != nil {
				c.OnYes()
			}
		case k.Key == keyboard.KeyEscape, k.Rune == 'n', k.Rune == 'N':
			if c.OnNo != nil {
				c.OnNo()
			}
		}
	}
	return true
}
