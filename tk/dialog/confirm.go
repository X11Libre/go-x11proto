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
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
)

// Confirm is a small modal yes/no dialog: a message (one or more lines) above
// two push buttons. Either click the buttons or use the keyboard — Enter / 'y'
// confirms (OnYes), Escape / 'n' cancels (OnNo). Like FilePicker it does not
// self-destroy; the owner closes it from the callbacks.
//
// Fill the embedded Window (Parent/X/Y/W/H) and Font before Init. Set Floating
// (with an optional Title) for a separate, window-manager-managed window. The
// button labels default to "Yes"/"No" (YesLabel/NoLabel to override).
type Confirm struct {
	tk_core.Window
	Font     *font.Font
	Keymap   *keyboard.Map
	Message  string
	YesLabel string
	NoLabel  string
	OnYes    func()
	OnNo     func()
	Floating bool
	Title    string

	gc       *tk_core.GC
	yes, no  *tk_widget.Button
	wmDelete base.ATOM
}

const (
	confirmBtnW = 72
	confirmBtnH = 24
	confirmGap  = 16
	confirmPad  = 10
)

// Init creates and maps the dialog plus its buttons, and takes focus.
func (c *Confirm) Init() error {
	if c.YesLabel == "" {
		c.YesLabel = "Yes"
	}
	if c.NoLabel == "" {
		c.NoLabel = "No"
	}
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
	if c.Floating {
		c.wmDelete, _ = c.Window.EnableWMDelete()
	}
	if err := c.Window.Map(); err != nil {
		return err
	}

	// two push buttons centred along the bottom
	total := 2*confirmBtnW + confirmGap
	x0 := (int(c.W) - total) / 2
	by := int(c.H) - confirmBtnH - confirmPad
	c.yes = c.mkButton(c.YesLabel, x0, by, func() {
		if c.OnYes != nil {
			c.OnYes()
		}
	})
	c.no = c.mkButton(c.NoLabel, x0+confirmBtnW+confirmGap, by, func() {
		if c.OnNo != nil {
			c.OnNo()
		}
	})
	if err := c.yes.Init(); err != nil {
		return err
	}
	if err := c.no.Init(); err != nil {
		return err
	}

	c.Focus()
	return c.Draw()
}

func (c *Confirm) mkButton(label string, x, y int, onPress func()) *tk_widget.Button {
	return &tk_widget.Button{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: c.Conn}, Parent: &c.Window,
			X: base.INT16(x), Y: base.INT16(y), W: confirmBtnW, H: confirmBtnH,
		},
		Label:         label,
		Font:          c.Font,
		OnButtonPress: onPress,
	}
}

// Focus requests the keyboard focus.
func (c *Confirm) Focus() {
	_ = rpc.SetInputFocus(c.Conn.X11Conn, 2 /*RevertToParent*/, c.XID, 0)
}

// Draw paints the message lines (the buttons paint themselves).
func (c *Confirm) Draw() error {
	if c.gc == nil {
		return nil
	}
	if err := c.ClearArea(0, 0, 0, 0, false); err != nil {
		return err
	}
	lh := c.Font.Height()
	y := confirmPad + c.Font.Ascent
	for _, line := range strings.Split(c.Message, "\n") {
		if err := c.PutText8(c.gc.XID, confirmPad, base.INT16(y), line); err != nil {
			return err
		}
		y += lh
	}
	return nil
}

// HandleWindowEvent keeps keyboard shortcuts working alongside the buttons.
func (c *Confirm) HandleWindowEvent(ev events.Event) bool {
	if tk_core.IsWMDelete(ev, c.wmDelete) { // window manager close = "No"
		if c.OnNo != nil {
			c.OnNo()
		}
		return true
	}
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
