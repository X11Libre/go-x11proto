package main

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/keyboard"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
)

// promptBox is a small modal text-input dialog centered in the editor: a title
// label above a single-line TextView. Enter accepts (invoking the callback with
// the text), Escape cancels.
type promptBox struct {
	win   tk_core.Window
	title *tk_widget.Label
	input *tk_widget.TextView
	gc    *tk_core.GC
}

const (
	promptW = 460
	promptH = 2*barH + 10
)

// askFilename pops up a modal prompt seeded with initial; onAccept runs with the
// entered text when the user presses Enter. Only one prompt is open at a time.
func (e *Editor) askFilename(title, initial string, onAccept func(string)) {
	e.closePrompt()

	px := (winW - promptW) / 2
	py := (winH - promptH) / 2
	p := &promptBox{}

	p.win = tk_core.Window{
		Drawable: tk_core.Drawable{Conn: e.tk},
		Parent:   &e.frame.Window,
		X:        base.INT16(px), Y: base.INT16(py),
		W: promptW, H: promptH,
		EventMask:      base.CARD32(event_mask.Exposure),
		SetBorderPixel: true,
		BorderPixel:    e.conn.DefaultBlackPixel(),
		BorderWidth:    1,
	}
	if err := p.win.Create(); err != nil {
		return
	}
	_ = p.win.Map()
	_ = p.win.Raise()

	p.title = &tk_widget.Label{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: e.tk}, Parent: &p.win,
			X: 0, Y: 0, W: promptW, H: barH,
			EventMask: base.CARD32(event_mask.Exposure),
		},
		Renderer: e.font,
		Gc:       e.statusGc.XID,
		Align:    tk_widget.AlignLeft,
		Text:     title,
	}
	if err := p.title.Init(); err != nil {
		return
	}

	p.input = &tk_widget.TextView{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: e.tk}, Parent: &p.win,
			X: 4, Y: barH, W: promptW - 8, H: barH,
		},
		Font: e.font,
	}
	// Enter accepts, Escape cancels; everything else is normal line editing.
	p.input.OnKey = func(k keyboard.Event) bool {
		switch k.Key {
		case keyboard.KeyEnter:
			text := p.input.Text()
			e.closePrompt()
			onAccept(text)
			return true
		case keyboard.KeyEscape:
			e.closePrompt()
			return true
		}
		return false
	}
	if err := p.input.Init(); err != nil {
		return
	}
	p.input.SetText(initial)
	p.input.Focus()

	e.prompt = p
}

// closePrompt tears down the active prompt and returns focus to the editor.
func (e *Editor) closePrompt() {
	if e.prompt == nil {
		return
	}
	_ = e.prompt.input.Destroy()
	_ = e.prompt.title.Destroy()
	_ = e.prompt.win.Destroy()
	e.prompt = nil
	e.tv.Focus()
}
