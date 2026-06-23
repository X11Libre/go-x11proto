package widget

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

// TextRenderer draws and measures a string for a Label. It decouples the widget
// from any particular font: an implementation may use X11 server fonts, a
// bitmap glyph set, scalable glyphs, etc. scale is an implementation-defined
// integer size (e.g. an integer zoom for a bitmap font); renderers that have no
// notion of scale may ignore it.
type TextRenderer interface {
	// DrawText renders s onto d with gc, its top-left corner at (x, y).
	DrawText(d tk_core.Drawable, gc base.GC, x, y base.INT16, scale int, s string) error
	// Measure returns the pixel width and height s will occupy at scale.
	Measure(scale int, s string) (w, h int)
}

// Align selects the horizontal placement of a Label's text. The zero value is
// AlignCenter, preserving the original centred behaviour.
type Align int

const (
	AlignCenter Align = iota
	AlignLeft
	AlignRight
)

// Label is a window showing a single line of text drawn through a pluggable
// TextRenderer. The text is centred by default; set Align to AlignLeft (e.g.
// for a status/filename line) or AlignRight. With Transparent set it gives
// itself a ParentRelative background, so the parent's content shows through the
// gaps between glyphs — an overlay that paints only the text.
//
// The embedded Window's W/H/X/Y/Parent/EventMask must be filled in before Init
// (EventMask must include Exposure for the label to repaint itself). The caller
// owns the Gc.
type Label struct {
	tk_core.Window
	Text        string
	Scale       int
	Gc          base.GC
	Renderer    TextRenderer
	Transparent bool
	Align       Align
	PadX        int // left/right inset for AlignLeft/AlignRight (default 2)
}

// Init creates and maps the label. It installs the label as its own window
// handler, so it repaints on Expose.
func (l *Label) Init() error {
	l.Window.SetWindowHandler(l)
	if err := l.Window.Create(); err != nil {
		return err
	}
	if l.Transparent {
		if err := l.Window.SetBackgroundPixmap(tk_core.ParentRelative); err != nil {
			return err
		}
	}
	return l.Window.Map()
}

// SetText changes the displayed text and repaints.
func (l *Label) SetText(s string) error {
	l.Text = s
	return l.Draw()
}

// Draw clears the window and paints the text according to Align. It is called
// on Expose, but may also be called directly.
func (l *Label) Draw() error {
	if err := l.ClearArea(0, 0, 0, 0, false); err != nil {
		return err
	}
	if l.Renderer == nil || l.Text == "" {
		return nil
	}
	tw, th := l.Renderer.Measure(l.Scale, l.Text)
	x := alignX(l.Align, int(l.W), tw, l.PadX)
	y := (int(l.H) - th) / 2
	return l.Renderer.DrawText(l.Drawable, l.Gc, base.INT16(x), base.INT16(y), l.Scale, l.Text)
}

// alignX computes the text's left x for the given alignment in a window of
// width winW, for text of width textW. pad (defaulting to 2) is the inset used
// by the left/right alignments.
func alignX(a Align, winW, textW, pad int) int {
	if pad == 0 {
		pad = 2
	}
	switch a {
	case AlignLeft:
		return pad
	case AlignRight:
		return winW - textW - pad
	default: // AlignCenter
		return (winW - textW) / 2
	}
}

// HandleWindowEvent repaints the label on Expose.
func (l *Label) HandleWindowEvent(ev events.Event) bool {
	if _, ok := ev.(*events.ExposeEvent); ok {
		l.Draw()
	}
	return true
}
