package main

import (
	tetris_font "github.com/X11Libre/go-x11proto/demo/tetris64/font"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

var helpLines = []string{
	"ARROWS/HJKL  MOVE",
	"DOWN / J     SOFT DROP",
	"UP / K       ROTATE",
	"ENTER        HARD DROP",
	"SPACE        PAUSE",
	"C            COLOR/MONO",
	"G            GHOST",
	"F            FULLSCREEN",
	"+ / -        RESOLUTION",
	"Q            QUIT",
}

// HelpWindow is the controls/help page rendered as its own window (a tk Window),
// shown and hidden by mapping/unmapping rather than repainting the background.
// Its thick light border mirrors the play-field border baked into the art.
type HelpWindow struct {
	tk_core.Window
	gcText   base.GC
	gcBlack  base.GC
	gcBorder base.GC // play-field-coloured border (matches the well frame)
	fs       int     // font scale
	resH     int     // framebuffer height, for the vertical layout proportions
}

// newHelpWindow creates the (initially unmapped) help window as a child of
// parent at (x,y) sized w x h in parent coordinates, with a black background.
func newHelpWindow(tk *tk_core.TkConn, parent base.WINDOW, x, y, w, h int, gcText, gcBlack, gcBorder base.GC, fs, resH int) (*HelpWindow, error) {
	hw := &HelpWindow{
		Window: tk_core.Window{
			Drawable:  tk_core.Drawable{Conn: tk},
			ParentXID: parent,
			X:         base.INT16(x),
			Y:         base.INT16(y),
			W:         base.CARD16(w),
			H:         base.CARD16(h),
			// only expose - key events propagate up to the frame's handler so
			// F1 / Shift+H still close the page while it is on top.
			EventMask: event_mask.Exposure,
			// black interior + black (invisible) outer border; the visible
			// border is the thin coloured frame drawn in Draw.
			SetBackPixel:   true,
			BackPixel:      0,
			SetBorderPixel: true,
			BorderPixel:    0,
		},
		gcText:   gcText,
		gcBlack:  gcBlack,
		gcBorder: gcBorder,
		fs:       fs,
		resH:     resH,
	}
	if err := hw.Create(); err != nil {
		return nil, err
	}
	hw.SetWindowHandler(hw)
	return hw, nil
}

// HandleWindowEvent redraws the page on expose.
func (hw *HelpWindow) HandleWindowEvent(ev events.Event) bool {
	if _, ok := ev.(*events.ExposeEvent); ok {
		hw.Draw()
	}
	return true
}

// Draw paints the border and control list. Coordinates are window-relative; the
// vertical positions keep the same C64-space proportions as the play field.
func (hw *HelpWindow) Draw() {
	W, H := int(hw.W), int(hw.H)
	fs := hw.fs
	bt := fs / 2 // thin line, like the play-field border in the art
	if bt < 2 {
		bt = 2
	}
	indent := W / 15
	d := hw.Drawable // populated by Create(); draw via the tk Drawable methods

	// black interior + thin play-field-coloured border
	d.FillRects(hw.gcBlack, []base.Rectangle{
		{X: 0, Y: 0, Width: base.CARD16(W), Height: base.CARD16(H)},
	})
	d.FillRects(hw.gcBorder, []base.Rectangle{
		{X: 0, Y: 0, Width: base.CARD16(W), Height: base.CARD16(bt)},
		{X: 0, Y: base.INT16(H - bt), Width: base.CARD16(W), Height: base.CARD16(bt)},
		{X: 0, Y: 0, Width: base.CARD16(bt), Height: base.CARD16(H)},
		{X: base.INT16(W - bt), Y: 0, Width: base.CARD16(bt), Height: base.CARD16(H)},
	})

	tetris_font.DrawString(d, hw.gcText,
		base.INT16((W-4*8*fs)/2), base.INT16(12*hw.resH/200), fs, "HELP")

	sepY := 20 * hw.resH / 200
	d.FillRects(hw.gcBorder, []base.Rectangle{
		{X: base.INT16(indent), Y: base.INT16(sepY), Width: base.CARD16(W - 2*indent), Height: base.CARD16(bt)},
	})

	for i, line := range helpLines {
		yy := (30 + i*10) * hw.resH / 200
		tetris_font.DrawString(d, hw.gcText, base.INT16(indent), base.INT16(yy), fs, line)
	}

	tetris_font.DrawString(d, hw.gcText,
		base.INT16((W-19*8*fs)/2), base.INT16(138*hw.resH/200), fs, "PRESS F1 OR SHIFT+H")
}
