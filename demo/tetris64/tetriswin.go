package main

import (
	"fmt"

	tetris_font "github.com/X11Libre/go-x11proto/demo/tetris64/font"
	"github.com/X11Libre/go-x11proto/demo/tetris64/game"
	"github.com/X11Libre/go-x11proto/demo/tetris64/sidplayer"
	"github.com/X11Libre/go-x11proto/proto/base"
	proto_core "github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/core/keycodes"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
)

type TetrisWin struct {
	conn   *proto_core.X11Conn
	tkConn *tk_core.TkConn

	frameWin tk_core.Window // top-level resizable frame (black bg)
	bgWin    tk_core.Window // child of frame, holds bg pixmap, centered
	boardWin tk_core.Window // child of bgWin, holds board rendering

	gcText        *tk_core.GC
	gcBlack       *tk_core.GC
	gcBorder      *tk_core.GC // play-field-coloured border for the help page
	gcBorderTheme string      // theme gcBorder was sampled for
	gcColors      map[uint8]*tk_core.GC
	gcGhost       map[uint8]*tk_core.GC
	gcShade       map[uint8]*tk_core.GC

	bg       base.PIXMAP
	loaderBg base.PIXMAP

	help     *HelpWindow      // controls overlay, mapped when showHelp
	gameOver *tk_widget.Label // "GAME OVER" overlay, present only when w.gs.GameOver

	digits    *tetris_font.Digits // score/lines digit font, rebuilt per resolution
	glyphTint [3]byte             // digit colour, sampled from the background art

	layout     resLayout
	scale      int
	resIdx     int
	fullscreen bool
	showHelp   bool
	showGhost  bool
	frameW     int
	frameH     int
	fbW        int // framebuffer (8:5 background) width; border = (frameW-fbW)/2

	gs     *game.State // the running game; created by startGame
	state  appState    // intro vs playing
	paused bool

	music *sidplayer.Player
}

func (w *TetrisWin) createWin(screenW, screenH int) {
	res := resolutions[w.resIdx]
	w.scale = res.scale
	w.layout = layouts[w.resIdx]
	w.showGhost = true
	l := w.layout

	// The framebuffer keeps the original C64 8:5 aspect (320x200, zoomed). The
	// background art is cropped to it; the black left/right border is generated
	// by the frame window's black background, not baked into the pixmap.
	w.fbW = res.h * 8 / 5
	border := (res.w - w.fbW) / 2

	// digit colour matches the background's score text (sampled from the FHD art)
	w.glyphTint = glyphTintColor()
	w.bg = w.uploadBg(loadFrame(), w.fbW, res.h)
	w.loaderBg = w.uploadBg(loadLoader(), w.fbW, res.h)

	// top-level frame window (black background, resizable)
	w.frameWin = tk_core.Window{
		Drawable: tk_core.Drawable{
			Conn: w.tkConn,
		},
		Parent:       w.tkConn.GetRoot(),
		Name:         fmt.Sprintf("C64 TETRIS - %s / %s", res.name, theme),
		X:            base.INT16((screenW - res.w) / 2),
		Y:            base.INT16((screenH - res.h) / 2),
		W:            base.CARD16(res.w),
		H:            base.CARD16(res.h),
		EventMask:    0xFFFFFF,
		SetBackPixel: true, // black background (the border around the framebuffer)
		BackPixel:    0,
	}
	w.frameWin.SetWindowHandler(w)
	errPanic(w.frameWin.Create(), "create frame")

	// bg window: child of frame, framebuffer-sized (8:5), centered so the
	// frame's black background shows through as the left/right border.
	w.bgWin = tk_core.Window{
		Drawable:  tk_core.Drawable{Conn: w.tkConn},
		ParentXID: w.frameWin.XID,
		X:         base.INT16(border),
		Y:         0,
		W:         base.CARD16(w.fbW),
		H:         base.CARD16(res.h),
		EventMask: 0xFFFFFF,
	}
	w.bgWin.SetWindowHandler(w)
	errPanic(w.bgWin.Create(), "create bg window")
	w.bgWin.SetBackgroundPixmap(w.loaderBg)

	// board window: child of bg, at board position, black bg
	bw := 10 * l.cell
	bh := 21 * l.cell
	w.boardWin = tk_core.Window{
		Drawable:     tk_core.Drawable{Conn: w.tkConn},
		ParentXID:    w.bgWin.XID,
		X:            base.INT16(l.bx),
		Y:            base.INT16(l.by),
		W:            base.CARD16(bw),
		H:            base.CARD16(bh),
		EventMask:    0xFFFFFF,
		SetBackPixel: true,
		BackPixel:    0,
	}
	w.boardWin.SetWindowHandler(w)
	errPanic(w.boardWin.Create(), "create board window")

	w.frameW = res.w
	w.frameH = res.h

	// build the greyscale digit glyphs scaled for this resolution
	if w.digits == nil {
		w.digits = tetris_font.NewDigits(w.tkConn, loadGlyphMasters())
	}
	w.digits.Build(w.layout.adv, w.scale, w.glyphTint)

	// help overlay: its own window above the play area, same box geometry the
	// drawn-on-bg version used. Created unmapped (shown on demand).
	w.ensureGCs()
	if w.gcBorder == nil || w.gcBorderTheme != theme {
		if w.gcBorder != nil {
			w.gcBorder.Free() // theme changed: drop the old border colour
		}
		bc := wellBorderColor() // match the play-field border baked into the art
		w.gcBorder, _ = w.tkConn.CreateGC1(
			base.CARD32(uint32(bc[0])<<16|uint32(bc[1])<<8|uint32(bc[2])), 0x000000, 0)
		w.gcBorderTheme = theme
	}
	hx := 40 * w.fbW / 320
	hy := 24 * res.h / 200
	help, err := newHelpWindow(w.tkConn, w.frameWin.XID,
		border+hx, hy, w.fbW-2*hx, res.h-2*hy, w.gcText.XID, w.gcBlack.XID, w.gcBorder.XID, w.scale, res.h)
	errPanic(err, "create help window")
	w.help = help

	// map child windows first
	w.bgWin.Map()
	if w.state == statePlaying {
		w.boardWin.Map()
	}
	if w.showHelp {
		w.help.Map()
		w.help.Draw()
	}
	errPanic(w.frameWin.Map(), "map frame")
}

// ---- fullscreen toggle via _NET_WM_STATE ----

func (w *TetrisWin) toggleFullscreen() {
	w.fullscreen = !w.fullscreen

	wmState, err := w.tkConn.InternAtom("_NET_WM_STATE")
	if err != nil {
		return
	}
	fsAtom, err := w.tkConn.InternAtom("_NET_WM_STATE_FULLSCREEN")
	if err != nil {
		return
	}

	ev := events.ClientMessageEvent{
		Window:      w.frameWin.XID,
		MessageType: wmState,
		Format:      32,
		// _NET_WM_STATE: action = toggle (2), property = _NET_WM_STATE_FULLSCREEN
		Data: [5]base.CARD32{2, base.CARD32(fsAtom), 0, 0, 0},
	}

	w.conn.Send(&request.SendEventRequest{
		Propagate:   false,
		Destination: w.conn.DefaultRoot(),
		EventMask:   0xFFFFFF,
		Event:       ev.Encode(w.conn.BE),
	})
}

// toggleHelp shows/hides the help window (F1 or Shift+H). It pauses the game by
// halting gravity (see gameLoop). In-game the board is hidden while help is up
// so the well does not peek past the box; closing just unmaps the help window,
// revealing the untouched background beneath.
func (w *TetrisWin) toggleHelp() {
	w.showHelp = !w.showHelp
	if w.showHelp {
		// hide the board so the well doesn't peek past the help box
		if w.state == statePlaying {
			w.boardWin.Unmap()
		}
		w.help.Map()
		w.help.Draw()
	} else {
		w.help.Unmap()
		if w.state == statePlaying {
			w.boardWin.Map()
			w.drawGame()
		}
	}
}

// recreateWindows tears down and rebuilds the windows for the current
// resolution/theme, reloading the backgrounds and rescaling the glyphs.
func (w *TetrisWin) recreateWindows() {
	screen := w.conn.Setup.Screens[0]
	screenW := int(screen.Width)
	screenH := int(screen.Height)

	// destroy all 3 windows (children before parent); the GAME OVER label is a
	// child of bgWin and goes with it, so just drop our stale reference.
	w.gameOver = nil
	w.boardWin.Destroy()
	w.bgWin.Destroy()
	w.frameWin.Destroy()
	w.conn.SendAndWait(&request.GetInputFocusRequest{}) // round-trip barrier

	// createWin reassigns fresh Window structs (zero XID), so no reset needed.
	w.createWin(screenW, screenH)

	switch w.state {
	case stateIntro:
		w.drawIntro()
	case statePlaying:
		w.bgWin.SetBackgroundPixmap(w.bg)
		w.bgWin.ClearArea(0, 0, 0, 0, false)
		w.drawGame()
	}
}

func (w *TetrisWin) cycleRes(dir int) {
	n := len(resolutions)
	w.resIdx = (w.resIdx + dir + n) % n
	w.recreateWindows()
}

func (w *TetrisWin) toggleTheme() {
	if theme == "color" {
		theme = "mono"
	} else {
		theme = "color"
	}
	cachedGlyphTint = nil // re-sample the digit colour from the new theme's art
	cachedBorder = nil    // and the play-field border colour
	w.recreateWindows()
}

func (w *TetrisWin) shadeGC(color uint8) *tk_core.GC {
	if w.gcShade == nil {
		w.gcShade = make(map[uint8]*tk_core.GC)
	}
	if g, ok := w.gcShade[color]; ok {
		return g
	}
	rgba := game.PieceColors[game.PieceType(color)]
	g, _ := w.tkConn.CreateGC1(base.CARD32(shadeColor(rgba)), 0x000000, 0)
	w.gcShade[color] = g
	return g
}

// scanlineRects returns the darker horizontal stripes (every other ~t rows)
// for a set of cell rectangles, producing a subtle scanline texture.
func (w *TetrisWin) scanlineRects(rects []base.Rectangle) []base.Rectangle {
	t := w.scale / 4
	if t < 1 {
		t = 1
	}
	cell := w.layout.cell
	var out []base.Rectangle
	for _, r := range rects {
		for yy := t; yy < cell; yy += 2 * t {
			h := t
			if yy+h > cell {
				h = cell - yy
			}
			out = append(out, base.Rectangle{
				X: r.X, Y: r.Y + base.INT16(yy),
				Width: r.Width, Height: base.CARD16(h),
			})
		}
	}
	return out
}

func (w *TetrisWin) gcCol(color uint8) (full, ghost *tk_core.GC) {
	if w.gcColors == nil {
		w.gcColors = make(map[uint8]*tk_core.GC)
		w.gcGhost = make(map[uint8]*tk_core.GC)
	}
	var ok bool
	if full, ok = w.gcColors[color]; ok {
		ghost = w.gcGhost[color]
		return
	}
	rgba := game.PieceColors[game.PieceType(color)]
	full, _ = w.tkConn.CreateGC1(base.CARD32(rgba), 0x000000, 0)
	ghost, _ = w.tkConn.CreateGC1(base.CARD32(dimColor(rgba)), 0x000000, 0)
	w.gcColors[color] = full
	w.gcGhost[color] = ghost
	return
}

// uploadBg decodes a background PNG and uploads it as a server-side pixmap of
// dstW x dstH, upscaling (Catmull-Rom) from the embedded FHD source when the
// target resolution is larger.
func (w *TetrisWin) uploadBg(data []byte, dstW, dstH int) base.PIXMAP {
	img, err := decodeScaled(data, dstW, dstH)
	errPanic(err, "decode image")
	pm, err := img.Upload(w.conn, w.conn.DefaultRoot())
	errPanic(err, "upload pixmap")
	return pm
}

// showGameOver lazily creates a transparent "GAME OVER" label centred over the
// board (above boardWin, which would otherwise hide text drawn on bgWin). It is
// a child of bgWin, so a window teardown (recreateWindows) destroys it too.
func (w *TetrisWin) showGameOver() {
	if w.gameOver != nil {
		return
	}
	const text = "GAME OVER"
	tw, th := tetris_font.Renderer{}.Measure(w.scale, text)
	l := w.layout
	lbl := &tk_widget.Label{
		Window: tk_core.Window{
			Drawable:  tk_core.Drawable{Conn: w.tkConn},
			ParentXID: w.bgWin.XID,
			X:         base.INT16(l.gameOverX),
			Y:         base.INT16(l.gameOverY),
			W:         base.CARD16(tw),
			H:         base.CARD16(th),
			EventMask: event_mask.Exposure,
		},
		Text:        text,
		Scale:       w.scale,
		Gc:          w.gcText.XID,
		Renderer:    tetris_font.Renderer{},
		Transparent: true,
	}
	if err := lbl.Init(); err != nil {
		return
	}
	w.gameOver = lbl
}

// hideGameOver removes the GAME OVER overlay if present.
func (w *TetrisWin) hideGameOver() {
	if w.gameOver != nil {
		w.gameOver.Destroy()
		w.gameOver = nil
	}
}

func (w *TetrisWin) drawIntro() {
	w.bgWin.SetBackgroundPixmap(w.loaderBg)
	w.bgWin.ClearArea(0, 0, 0, 0, false)
}

// startGame leaves the intro for a fresh game: swap to the play background,
// reset state, show the board and paint.
func (w *TetrisWin) startGame() {
	w.state = statePlaying
	w.paused = false
	w.gs = game.New()
	w.bgWin.SetBackgroundPixmap(w.bg)
	w.bgWin.ClearArea(0, 0, 0, 0, false)
	w.boardWin.Map()
	w.drawGame()
}

// restart resets the game after game over and repaints.
func (w *TetrisWin) restart() {
	w.gs.Reset()
	w.paused = false
	w.drawGame()
}

// ensureGCs lazily creates the black + washed-out-grey GCs used for text and
// fills (the grey #BABABA matches the original C64 readout, not stark white).
func (w *TetrisWin) ensureGCs() {
	if w.gcBlack == nil {
		gcid, err := w.tkConn.CreateGC1(0x000000, 0x000000, 0)
		errPanic(err, "CreateGC1 black")
		w.gcBlack = gcid
	}
	if w.gcText == nil {
		gcid, err := w.tkConn.CreateGC1(0xBABABA, 0x000000, 0)
		errPanic(err, "CreateGC1 text")
		w.gcText = gcid
	}
}

// pieceCells returns the top-left pixel position of each occupied block of p,
// with the piece's (0,0) cell at (originX, originY). Blocks above the top edge
// (negative Y) are skipped. Shared by the solid-fill and ghost-outline drawers.
func (w *TetrisWin) pieceCells(p game.Piece, originX, originY int) []base.Point {
	cell := w.layout.cell
	var cells []base.Point
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if p.Get(x, y) == 0 {
				continue
			}
			py := originY + y*cell
			if py < 0 {
				continue
			}
			cells = append(cells, base.Point{X: base.INT16(originX + x*cell), Y: base.INT16(py)})
		}
	}
	return cells
}

// fillPiece draws piece p as solid blocks (l.cell square) onto target, with the
// piece's (0,0) cell at (originX, originY), plus the scanline shading. Used for
// both the falling piece on the board and the next-piece preview.
func (w *TetrisWin) fillPiece(target tk_core.Window, p game.Piece, originX, originY int) {
	cell := base.CARD16(w.layout.cell)
	var rects []base.Rectangle
	for _, c := range w.pieceCells(p, originX, originY) {
		rects = append(rects, base.Rectangle{X: c.X, Y: c.Y, Width: cell, Height: cell})
	}
	full, _ := w.gcCol(uint8(p.Type))
	target.FillRects(full.XID, rects)
	target.FillRects(w.shadeGC(uint8(p.Type)).XID, w.scanlineRects(rects))
}

// ghostPiece draws p as a hollow outline (the landing preview) onto target, in
// the piece's dimmed ghost colour, at the same origin convention as fillPiece.
func (w *TetrisWin) ghostPiece(target tk_core.Window, p game.Piece, originX, originY int) {
	cell := w.layout.cell
	bt := cell / 8 // outline thickness
	if bt < 1 {
		bt = 1
	}
	cw, t := base.CARD16(cell), base.CARD16(bt)
	var rects []base.Rectangle
	for _, c := range w.pieceCells(p, originX, originY) {
		rects = append(rects,
			base.Rectangle{X: c.X, Y: c.Y, Width: cw, Height: t},                       // top
			base.Rectangle{X: c.X, Y: c.Y + base.INT16(cell-bt), Width: cw, Height: t}, // bottom
			base.Rectangle{X: c.X, Y: c.Y, Width: t, Height: cw},                       // left
			base.Rectangle{X: c.X + base.INT16(cell-bt), Y: c.Y, Width: t, Height: cw}, // right
		)
	}
	_, ghost := w.gcCol(uint8(p.Type))
	target.FillRects(ghost.XID, rects)
}

func (w *TetrisWin) drawGame() {
	w.ensureGCs()

	fs := w.scale
	l := w.layout

	w.bgWin.SetBackgroundPixmap(w.bg)

	// clear board area (coordinates relative to boardWin)
	w.boardWin.FillRects(w.gcBlack.XID, []base.Rectangle{{
		X: 0, Y: 0,
		Width: base.CARD16(10 * l.cell), Height: base.CARD16(21 * l.cell),
	}})

	// clear next-piece preview area
	w.bgWin.FillRects(w.gcBlack.XID, []base.Rectangle{{
		X: base.INT16(l.nx), Y: base.INT16(l.ny),
		Width: base.CARD16(4 * l.cell), Height: base.CARD16(4 * l.cell),
	}})

	// clear score/lines text areas (from the baked number's left edge to numRight)
	w.bgWin.FillRects(w.gcBlack.XID, []base.Rectangle{
		{X: base.INT16(l.scoreX), Y: base.INT16(l.scoreY), Width: base.CARD16(l.numRight - l.scoreX), Height: base.CARD16(8 * fs)},
		{X: base.INT16(l.linesX), Y: base.INT16(l.linesY), Width: base.CARD16(l.numRight - l.linesX), Height: base.CARD16(8 * fs)},
	})

	// draw board cells (relative to boardWin)
	byColor := make(map[uint8][]base.Rectangle)
	for y := 0; y < 21; y++ {
		for x := 0; x < 10; x++ {
			c := w.gs.Board[y][x]
			if c == 0 {
				continue
			}
			byColor[c] = append(byColor[c], base.Rectangle{
				X: base.INT16(x * l.cell), Y: base.INT16(y * l.cell),
				Width: base.CARD16(l.cell), Height: base.CARD16(l.cell),
			})
		}
	}
	for c, rects := range byColor {
		full, _ := w.gcCol(c)
		w.boardWin.FillRects(full.XID, rects)
		w.boardWin.FillRects(w.shadeGC(c).XID, w.scanlineRects(rects))
	}

	// ghost piece (landing preview, relative to boardWin)
	dist := w.gs.DropDistance()
	if w.showGhost && dist > 0 {
		w.ghostPiece(w.boardWin, w.gs.Current, w.gs.CX*l.cell, (w.gs.CY+dist)*l.cell)
	}

	// current piece (relative to boardWin)
	w.fillPiece(w.boardWin, w.gs.Current, w.gs.CX*l.cell, w.gs.CY*l.cell)

	// next piece preview (relative to bgWin), centred in its 4x4 box
	next := game.NewPiece(w.gs.Next)
	mx, my, mx2, my2 := next.Bounds()
	offX := (4 - (mx2 - mx + 1)) / 2
	offY := (4 - (my2 - my + 1)) / 2
	w.fillPiece(w.bgWin, next, l.nx+(offX-mx)*l.cell, l.ny+(offY-my)*l.cell)

	// score/lines/level use the greyscale digit pixmaps (scaled from masters)
	// right-align the numbers to numRight
	scoreStr := fmt.Sprintf("%05d", w.gs.Score)
	linesStr := fmt.Sprintf("%03d", w.gs.Lines)
	w.digits.Draw(w.bgWin.XID, w.gcText.XID, base.INT16(l.numRight-len(scoreStr)*l.adv), base.INT16(l.scoreY), scoreStr)
	w.digits.Draw(w.bgWin.XID, w.gcText.XID, base.INT16(l.numRight-len(linesStr)*l.adv), base.INT16(l.linesY), linesStr)

	// GAME OVER: a transparent overlay label above the board (see showGameOver)
	if w.gs.GameOver {
		w.showGameOver()
	} else {
		w.hideGameOver()
	}

	if w.paused {
		// overlay "PAUSE" centred on the board (boardWin sits on top)
		boardDraw := w.boardWin.Drawable
		const msg = "PAUSE"
		tw := len(msg) * 8 * fs
		px := (10*l.cell - tw) / 2
		py := 10*l.cell - 4*fs
		w.boardWin.FillRects(w.gcBlack.XID, []base.Rectangle{
			{X: base.INT16(px - fs), Y: base.INT16(py - fs), Width: base.CARD16(tw + 2*fs), Height: base.CARD16(8*fs + 2*fs)},
		})
		tetris_font.DrawString(boardDraw, w.gcText.XID, base.INT16(px), base.INT16(py), fs, msg)
	}
}

// HandleWindowEvent is the tk WindowHandler for the frame and its subwindows
// (all registered via tk.Window.SetWindowHandler), so a dedicated
// HandleX11WindowEvent on TetrisWin is no longer needed.
func (w *TetrisWin) HandleWindowEvent(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		switch w.state {
		case stateIntro:
			w.drawIntro()
		case statePlaying:
			w.drawGame()
		}
	case *events.ConfigureEvent:
		if e.TargetWindow != w.frameWin.XID {
			return false
		}
		nw := int(e.Width)
		nh := int(e.Height)
		if nw != w.frameW || nh != w.frameH {
			w.frameW = nw
			w.frameH = nh

			// auto-switch to a larger resolution when window grows
			bestIdx := w.resIdx
			for i, r := range resolutions {
				if r.w <= nw && r.h <= nh {
					bestIdx = i
				}
			}
			if bestIdx > w.resIdx {
				w.cycleRes(bestIdx - w.resIdx)
				return true
			}

			// re-center bgWin (framebuffer-sized; black frame is the border)
			res := resolutions[w.resIdx]
			bgX := (nw - w.fbW) / 2
			bgY := (nh - res.h) / 2
			if bgX < 0 {
				bgX = 0
			}
			if bgY < 0 {
				bgY = 0
			}
			w.bgWin.Move(base.INT16(bgX), base.INT16(bgY))
		}
	case *events.KeyPressEvent:
		// Shift+H toggles help as well as F1 (plain h is vi-move-left, below).
		if e.Detail == keycodes.H && e.State&keycodes.ShiftMask != 0 {
			w.toggleHelp()
			return true
		}
		switch e.Detail {
		case keycodes.Q:
			close(doneCh)
			return true
		case keycodes.F:
			w.toggleFullscreen()
			return true
		case keycodes.F1: // toggle help (pauses the game); works in loader and game
			w.toggleHelp()
			return true
		case keycodes.G:
			w.showGhost = !w.showGhost
			w.drawGame()
			return true
		case keycodes.C: // switch colour <-> mono theme
			w.toggleTheme()
			return true
		case keycodes.PlusMain, keycodes.PlusKP, keycodes.PlusEqual:
			w.cycleRes(+1)
			return true
		case keycodes.MinusMain, keycodes.MinusKP, keycodes.MinusUS:
			w.cycleRes(-1)
			return true
		}

		// while the help page is open the game is w.paused: ignore everything else
		if w.showHelp {
			return true
		}

		if w.state == stateIntro {
			w.startGame()
			return true
		}

		if e.Detail == keycodes.Return && w.gs.GameOver { // restart after game over
			w.restart()
			return true
		}

		if !w.gs.GameOver {
			switch e.Detail {
			case keycodes.Space: // pause / resume
				w.paused = !w.paused
			case keycodes.Left, keycodes.H:
				if !w.paused {
					w.gs.MoveLeft()
				}
			case keycodes.Right, keycodes.L:
				if !w.paused {
					w.gs.MoveRight()
				}
			case keycodes.Down, keycodes.J:
				if !w.paused {
					w.gs.MoveDown()
				}
			case keycodes.Up, keycodes.K:
				if !w.paused {
					w.gs.Rotate()
				}
			case keycodes.Return: // hard drop
				if !w.paused {
					w.gs.HardDrop()
				}
			}
			w.drawGame()
		}
	}
	return true
}
