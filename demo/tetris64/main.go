package main

import (
	"bytes"
	"embed"
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"time"

	tetris_font "github.com/X11Libre/go-x11proto/demo/tetris64/font"
	"github.com/X11Libre/go-x11proto/demo/tetris64/game"
	"github.com/X11Libre/go-x11proto/demo/tetris64/sidplayer"
	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/base"
	proto_core "github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/xpm"
)

// All background art is compiled into the binary: both themes, every
// resolution (the screenshots/ reference dir is intentionally left out).
//
//go:embed assets/color assets/mono
var bgFS embed.FS

//go:embed assets/music/tetris.sid
var sidData []byte

// theme selects the asset set; combined with the resolution name it forms the
// asset path prefix "<theme>/<res>/" (e.g. "color/FHD"). Currently fixed.
var theme = "color"

type appState int

const (
	stateIntro appState = iota
	statePlaying
)

var curState = stateIntro
var paused bool
var gs *game.State

type resOpt struct {
	w, h  int
	scale int
	name  string
}

var resolutions = []resOpt{
	{1920, 1080, 6, "FHD"},
	{2560, 1440, 8, "WQHD"},
	{3840, 2160, 12, "UHD4K"},
}

type resLayout struct {
	cell                 int
	bx, by               int
	nx, ny               int
	scoreX, scoreY       int
	linesX, linesY       int
	numRight             int // right edge the score/lines numbers are aligned to
	adv                  int // digit advance / cell width (narrower than 8*scale)
	gameOverX, gameOverY int
}

// Positions are relative to the framebuffer (bgWin) origin, i.e. the cropped
// background without the black side-border. X coords = original full-frame X
// minus the per-resolution border (FHD 96, WQHD 128, UHD4K 192).
var layouts = []resLayout{
	{cell: 42, bx: 659, by: 99, nx: 1302, ny: 174, scoreX: 1379, scoreY: 70, linesX: 1449, linesY: 119, numRight: 1643, adv: 48, gameOverX: 624, gameOverY: 475},
	{cell: 55, bx: 882, by: 143, nx: 1736, ny: 232, scoreX: 1895, scoreY: 94, linesX: 1952, linesY: 159, numRight: 2247, adv: 64, gameOverX: 832, gameOverY: 633},
	{cell: 83, bx: 1321, by: 212, nx: 2604, ny: 348, scoreX: 2842, scoreY: 141, linesX: 2933, linesY: 238, numRight: 3370, adv: 96, gameOverX: 1248, gameOverY: 950},
}

type TetrisWin struct {
	conn   *proto_core.X11Conn
	tkConn *tk_core.TkConn

	frameWin tk_core.Window // top-level resizable frame (black bg)
	bgWin    base.WINDOW    // child of frame, holds bg pixmap, centered
	boardWin base.WINDOW    // child of bgWin, holds board rendering

	gcText   base.GC
	gcBlack  base.GC
	gcColors map[uint8]base.GC
	gcGhost  map[uint8]base.GC
	gcShade  map[uint8]base.GC

	bg       base.PIXMAP
	loaderBg base.PIXMAP

	digitPix  [10]base.PIXMAP // digit glyphs, rebuilt per resolution
	glyphTint [3]byte         // digit colour, sampled from the background art

	layout     resLayout
	scale      int
	resIdx     int
	fullscreen bool
	showHelp   bool
	showGhost  bool
	frameW     int
	frameH     int
	fbW        int // framebuffer (8:5 background) width; border = (frameW-fbW)/2
}

func errPanic(e error, s string) {
	if e != nil {
		panic(s + ": " + e.Error())
	}
}

// ---- asset helpers ----

// loadFrame / loadLoader return the embedded background PNG for the current
// theme and resolution.
func loadFrame(resName string) []byte {
	d, _ := bgFS.ReadFile("assets/" + theme + "/" + resName + "/frame.png")
	return d
}

func loadLoader(resName string) []byte {
	d, _ := bgFS.ReadFile("assets/" + theme + "/" + resName + "/loader.png")
	return d
}

func decodeImage(data []byte) (*xpm.Image, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err == nil {
		b := img.Bounds()
		rgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
		return &xpm.Image{Width: b.Dx(), Height: b.Dy(), Data: rgba.Pix}, nil
	}
	return xpm.DecodeBytes(data)
}

// ---- endian-aware write helper ----

func putCARD32(buf []byte, v uint32, be bool) {
	if be {
		binary.BigEndian.PutUint32(buf, v)
	} else {
		binary.LittleEndian.PutUint32(buf, v)
	}
}

// ---- window creation / recreation ----

func (w *TetrisWin) uploadBg(data []byte) base.PIXMAP {
	img, err := decodeImage(data)
	errPanic(err, "decode image")
	pm, err := img.Upload(w.conn, w.conn.DefaultRoot())
	errPanic(err, "upload pixmap")
	return pm
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
	w.bg = w.uploadBg(loadFrame(res.name))
	w.loaderBg = w.uploadBg(loadLoader(res.name))

	// top-level frame window (black background, resizable)
	w.frameWin = tk_core.Window{
		Drawable: tk_core.Drawable{
			Conn: w.tkConn,
		},
		Parent:    w.tkConn.GetRoot(),
		Name:      fmt.Sprintf("C64 TETRIS - %s / %s", res.name, theme),
		X:         base.INT16((screenW - res.w) / 2),
		Y:         base.INT16((screenH - res.h) / 2),
		W:         base.CARD16(res.w),
		H:         base.CARD16(res.h),
		EventMask: 0xFFFFFF,
	}
	w.frameWin.SetWindowHandler(w)
	errPanic(w.frameWin.Create(), "create frame")
	// set black background pixel for the frame
	rpc.ChangeWindowAttributes(w.conn, &request.ChangeWindowAttributesRequest{
		Window:    w.frameWin.XID,
		ValueMask: request.CW_BACKGROUND_PIXEL,
		BackPixel: 0,
	})

	// bg window: child of frame, framebuffer-sized (8:5), centered so the
	// frame's black background shows through as the left/right border.
	bgXID := base.WINDOW(w.conn.NextResourceID())
	_, err := w.conn.Send(&request.CreateWindowRequest{
		Depth:     0,
		Wid:       bgXID,
		Parent:    w.frameWin.XID,
		X:         base.INT16(border),
		Y:         0,
		Width:     base.CARD16(w.fbW),
		Height:    base.CARD16(res.h),
		Border:    0,
		Class:     request.WindowClass_InputOutput,
		Visual:    0,
		ValueMask: request.CW_EVENT_MASK,
		EventMask: 0xFFFFFF,
	})
	errPanic(err, "create bg window")
	w.bgWin = bgXID
	w.conn.RegisterWindowHandler(base.XID(w.bgWin), w)
	rpc.SetWindowBackgroundPixmap(w.conn, w.bgWin, w.loaderBg)

	// board window: child of bg, at board position, black bg
	bw := 10 * l.cell
	bh := 21 * l.cell
	boardXID := base.WINDOW(w.conn.NextResourceID())
	_, err = w.conn.Send(&request.CreateWindowRequest{
		Depth:     0,
		Wid:       boardXID,
		Parent:    w.bgWin,
		X:         base.INT16(l.bx),
		Y:         base.INT16(l.by),
		Width:     base.CARD16(bw),
		Height:    base.CARD16(bh),
		Border:    0,
		Class:     request.WindowClass_InputOutput,
		Visual:    0,
		ValueMask: request.CW_BACKGROUND_PIXEL | request.CW_EVENT_MASK,
		BackPixel: 0,
		EventMask: 0xFFFFFF,
	})
	errPanic(err, "create board window")
	w.boardWin = boardXID
	w.conn.RegisterWindowHandler(base.XID(w.boardWin), w)

	w.frameW = res.w
	w.frameH = res.h

	// build the greyscale digit glyphs scaled for this resolution
	w.buildDigitPixmaps()

	// map child windows first
	rpc.MapWindow(w.conn, w.bgWin)
	if curState == statePlaying {
		rpc.MapWindow(w.conn, w.boardWin)
	}
	errPanic(w.frameWin.Map(), "map frame")
}

// ---- fullscreen toggle via _NET_WM_STATE ----

func (w *TetrisWin) toggleFullscreen() {
	w.fullscreen = !w.fullscreen

	wmState, err := rpc.InternAtom(w.conn, "_NET_WM_STATE")
	if err != nil {
		return
	}
	fsAtom, err := rpc.InternAtom(w.conn, "_NET_WM_STATE_FULLSCREEN")
	if err != nil {
		return
	}

	be := w.conn.BE
	var ev [32]byte
	ev[0] = 33 // ClientMessage
	ev[1] = 32 // format (32-bit data)
	putCARD32(ev[4:8], uint32(w.frameWin.XID), be)
	putCARD32(ev[8:12], uint32(wmState), be)
	putCARD32(ev[12:16], 2, be) // action = toggle
	putCARD32(ev[16:20], uint32(fsAtom), be)

	w.conn.Send(&request.SendEventRequest{
		Propagate:   false,
		Destination: w.conn.DefaultRoot(),
		EventMask:   0xFFFFFF,
		Event:       ev,
	})
}

// ---- game rendering ----

func dimColor(rgba uint32) uint32 {
	r := (rgba >> 16) & 0xFF
	g := (rgba >> 8) & 0xFF
	b := rgba & 0xFF
	r = r * 3 / 8
	g = g * 3 / 8
	b = b * 3 / 8
	return (r << 16) | (g << 8) | b
}

// shadeColor darkens a colour to ~13/16 for the scanline rows that give the
// stones the soft, slightly-striped look of the original CRT screenshots.
func shadeColor(rgba uint32) uint32 {
	r := ((rgba >> 16) & 0xFF) * 13 / 16
	g := ((rgba >> 8) & 0xFF) * 13 / 16
	b := (rgba & 0xFF) * 13 / 16
	return (r << 16) | (g << 8) | b
}

func (w *TetrisWin) shadeGC(color uint8) base.GC {
	if w.gcShade == nil {
		w.gcShade = make(map[uint8]base.GC)
	}
	if g, ok := w.gcShade[color]; ok {
		return g
	}
	rgba := game.PieceColors[game.PieceType(color)]
	g, _ := rpc.CreateGC1(w.conn, base.CARD32(shadeColor(rgba)), 0x000000, 0)
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

func (w *TetrisWin) gcCol(color uint8) (full, ghost base.GC) {
	if w.gcColors == nil {
		w.gcColors = make(map[uint8]base.GC)
		w.gcGhost = make(map[uint8]base.GC)
	}
	var ok bool
	if full, ok = w.gcColors[color]; ok {
		ghost = w.gcGhost[color]
		return
	}
	rgba := game.PieceColors[game.PieceType(color)]
	full, _ = rpc.CreateGC1(w.conn, base.CARD32(rgba), 0x000000, 0)
	ghost, _ = rpc.CreateGC1(w.conn, base.CARD32(dimColor(rgba)), 0x000000, 0)
	w.gcColors[color] = full
	w.gcGhost[color] = ghost
	return
}

func (w *TetrisWin) fillRects(target base.DRAWABLE, gc base.GC, rects []base.Rectangle) {
	if len(rects) == 0 {
		return
	}
	rpc.FillRects(w.conn, target, gc, rects)
}

func pieceBounds(p game.Piece) (minX, minY, maxX, maxY int) {
	minX, minY = 4, 4
	maxX, maxY = -1, -1
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if p.Get(x, y) != 0 {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	return
}

func (w *TetrisWin) drawIntro() {
	rpc.SetWindowBackgroundPixmap(w.conn, w.bgWin, w.loaderBg)
	rpc.ClearArea(w.conn, w.bgWin, 0, 0, 0, 0, false)
	if w.showHelp {
		w.drawHelp()
	}
}

// ensureGCs lazily creates the black + washed-out-grey GCs used for text and
// fills (the grey #BABABA matches the original C64 readout, not stark white).
func (w *TetrisWin) ensureGCs() {
	if w.gcBlack.Invalid() {
		gcid, err := rpc.CreateGC1(w.conn, 0x000000, 0x000000, 0)
		errPanic(err, "CreateGC1 black")
		w.gcBlack = gcid
	}
	if w.gcText.Invalid() {
		gcid, err := rpc.CreateGC1(w.conn, 0xBABABA, 0x000000, 0)
		errPanic(err, "CreateGC1 text")
		w.gcText = gcid
	}
}

func (w *TetrisWin) drawGame() {
	w.ensureGCs()

	fs := w.scale
	l := w.layout

	rpc.SetWindowBackgroundPixmap(w.conn, w.bgWin, w.bg)

	// clear board area (coordinates relative to boardWin)
	w.fillRects(w.boardWin, w.gcBlack, []base.Rectangle{{
		X: 0, Y: 0,
		Width: base.CARD16(10 * l.cell), Height: base.CARD16(21 * l.cell),
	}})

	// clear next-piece preview area
	w.fillRects(w.bgWin, w.gcBlack, []base.Rectangle{{
		X: base.INT16(l.nx), Y: base.INT16(l.ny),
		Width: base.CARD16(4 * l.cell), Height: base.CARD16(4 * l.cell),
	}})

	// clear score/lines text areas (from the baked number's left edge to numRight)
	w.fillRects(w.bgWin, w.gcBlack, []base.Rectangle{
		{X: base.INT16(l.scoreX), Y: base.INT16(l.scoreY), Width: base.CARD16(l.numRight - l.scoreX), Height: base.CARD16(8 * fs)},
		{X: base.INT16(l.linesX), Y: base.INT16(l.linesY), Width: base.CARD16(l.numRight - l.linesX), Height: base.CARD16(8 * fs)},
	})

	// draw board cells (relative to boardWin)
	byColor := make(map[uint8][]base.Rectangle)
	for y := 0; y < 21; y++ {
		for x := 0; x < 10; x++ {
			c := gs.Board[y][x]
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
		w.fillRects(w.boardWin, full, rects)
		w.fillRects(w.boardWin, w.shadeGC(c), w.scanlineRects(rects))
	}

	// ghost piece (relative to boardWin)
	dist := gs.DropDistance()
	if w.showGhost && dist > 0 {
		p := gs.Current
		_, ghost := w.gcCol(uint8(p.Type))
		bt := l.cell / 8 // outline thickness
		if bt < 1 {
			bt = 1
		}
		var rects []base.Rectangle
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				if p.Get(x, y) == 0 {
					continue
				}
				bx := gs.CX + x
				by := gs.CY + y + dist
				if by < 0 {
					continue
				}
				px := base.INT16(bx * l.cell)
				py := base.INT16(by * l.cell)
				c := base.CARD16(l.cell)
				t := base.CARD16(bt)
				// draw each landing cell as an outline (grid) instead of a fill
				rects = append(rects,
					base.Rectangle{X: px, Y: py, Width: c, Height: t},                         // top
					base.Rectangle{X: px, Y: py + base.INT16(l.cell-bt), Width: c, Height: t}, // bottom
					base.Rectangle{X: px, Y: py, Width: t, Height: c},                         // left
					base.Rectangle{X: px + base.INT16(l.cell-bt), Y: py, Width: t, Height: c}, // right
				)
			}
		}
		w.fillRects(w.boardWin, ghost, rects)
	}

	// current piece (relative to boardWin)
	p := gs.Current
	full, _ := w.gcCol(uint8(p.Type))
	var rects []base.Rectangle
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if p.Get(x, y) == 0 {
				continue
			}
			bx := gs.CX + x
			by := gs.CY + y
			if by < 0 {
				continue
			}
			rects = append(rects, base.Rectangle{
				X: base.INT16(bx * l.cell), Y: base.INT16(by * l.cell),
				Width: base.CARD16(l.cell), Height: base.CARD16(l.cell),
			})
		}
	}
	w.fillRects(w.boardWin, full, rects)
	w.fillRects(w.boardWin, w.shadeGC(uint8(p.Type)), w.scanlineRects(rects))

	// next piece preview (relative to bgWin)
	next := game.NewPiece(gs.Next)
	full, _ = w.gcCol(uint8(next.Type))
	mx, my, mx2, my2 := pieceBounds(next)
	mw := mx2 - mx + 1
	mh := my2 - my + 1
	offX := (4 - mw) / 2
	offY := (4 - mh) / 2
	var rectsN []base.Rectangle
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if next.Get(x, y) == 0 {
				continue
			}
			rectsN = append(rectsN, base.Rectangle{
				X:     base.INT16(l.nx + (x-mx+offX)*l.cell),
				Y:     base.INT16(l.ny + (y-my+offY)*l.cell),
				Width: base.CARD16(l.cell), Height: base.CARD16(l.cell),
			})
		}
	}
	w.fillRects(w.bgWin, full, rectsN)
	w.fillRects(w.bgWin, w.shadeGC(uint8(next.Type)), w.scanlineRects(rectsN))

	bgDraw := tk_core.Drawable{Conn: w.tkConn, XID: w.bgWin}

	// score/lines/level use the greyscale digit pixmaps (scaled from masters)
	// right-align the numbers to numRight
	scoreStr := fmt.Sprintf("%05d", gs.Score)
	linesStr := fmt.Sprintf("%03d", gs.Lines)
	w.drawNumber(w.bgWin, base.INT16(l.numRight-len(scoreStr)*l.adv), base.INT16(l.scoreY), scoreStr)
	w.drawNumber(w.bgWin, base.INT16(l.numRight-len(linesStr)*l.adv), base.INT16(l.linesY), linesStr)

	if gs.GameOver {
		tetris_font.DrawString(bgDraw, w.gcText,
			base.INT16(l.gameOverX), base.INT16(l.gameOverY), fs, "GAME OVER")
	}

	if paused {
		// overlay "PAUSE" centred on the board (boardWin sits on top)
		boardDraw := tk_core.Drawable{Conn: w.tkConn, XID: w.boardWin}
		const msg = "PAUSE"
		tw := len(msg) * 8 * fs
		px := (10*l.cell - tw) / 2
		py := 10*l.cell - 4*fs
		w.fillRects(w.boardWin, w.gcBlack, []base.Rectangle{
			{X: base.INT16(px - fs), Y: base.INT16(py - fs), Width: base.CARD16(tw + 2*fs), Height: base.CARD16(8*fs + 2*fs)},
		})
		tetris_font.DrawString(boardDraw, w.gcText, base.INT16(px), base.INT16(py), fs, msg)
	}

	if w.showHelp {
		w.drawHelp()
	}
}

// drawHelp draws the help overlay (controls) on bgWin, framed with a border in
// the style of the play-field border. Used from both the loader and the game.
func (w *TetrisWin) drawHelp() {
	w.ensureGCs()
	res := resolutions[w.resIdx]
	fs := w.scale
	bgDraw := tk_core.Drawable{Conn: w.tkConn, XID: w.bgWin}
	hx := 40 * w.fbW / 320
	hy := 24 * res.h / 200
	hbw := w.fbW - 2*hx
	hbh := res.h - 2*hy
	bt := fs // border thickness (like the well border)
	if bt < 2 {
		bt = 2
	}

	// dark interior + light border
	w.fillRects(w.bgWin, w.gcBlack, []base.Rectangle{
		{X: base.INT16(hx), Y: base.INT16(hy), Width: base.CARD16(hbw), Height: base.CARD16(hbh)},
	})
	w.fillRects(w.bgWin, w.gcText, []base.Rectangle{
		{X: base.INT16(hx), Y: base.INT16(hy), Width: base.CARD16(hbw), Height: base.CARD16(bt)},
		{X: base.INT16(hx), Y: base.INT16(hy + hbh - bt), Width: base.CARD16(hbw), Height: base.CARD16(bt)},
		{X: base.INT16(hx), Y: base.INT16(hy), Width: base.CARD16(bt), Height: base.CARD16(hbh)},
		{X: base.INT16(hx + hbw - bt), Y: base.INT16(hy), Width: base.CARD16(bt), Height: base.CARD16(hbh)},
	})

	titleX := (w.fbW - 4*8*fs) / 2
	tetris_font.DrawString(bgDraw, w.gcText,
		base.INT16(titleX), base.INT16(hy+12*res.h/200), fs, "HELP")
	sepY := hy + 20*res.h/200
	w.fillRects(w.bgWin, w.gcText, []base.Rectangle{
		{X: base.INT16(hx + 16*w.fbW/320), Y: base.INT16(sepY), Width: base.CARD16(hbw - 32*w.fbW/320), Height: base.CARD16(bt)},
	})

	helpLines := []string{
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
	for i, line := range helpLines {
		yy := hy + (30+i*10)*res.h/200
		tetris_font.DrawString(bgDraw, w.gcText,
			base.INT16(hx+16*w.fbW/320), base.INT16(yy), fs, line)
	}

	closeX := (w.fbW - 17*8*fs) / 2
	tetris_font.DrawString(bgDraw, w.gcText,
		base.INT16(closeX), base.INT16(hy+138*res.h/200), fs, "PRESS F1 TO CLOSE")
}

// ---- event handler ----

func (w *TetrisWin) HandleX11WindowEvent(_ base.WINDOW, ev events.Event) bool {
	return w.HandleWindowEvent(ev)
}

func (w *TetrisWin) HandleWindowEvent(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		switch curState {
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
			w.conn.Send(&request.ConfigureWindowRequest{
				Window:    w.bgWin,
				ValueMask: request.CONFIG_WINDOW_X | request.CONFIG_WINDOW_Y,
				X:         base.INT16(bgX),
				Y:         base.INT16(bgY),
			})
		}
	case *events.KeyPressEvent:
		switch e.Detail {
		case 24: // q / Q
			close(doneCh)
			return true
		case 41: // f / F
			w.toggleFullscreen()
			return true
		case 67: // F1 - toggle help (pauses the game); works in loader and game
			w.showHelp = !w.showHelp
			if curState == statePlaying {
				if w.showHelp {
					// hide the board so the help overlay isn't occluded by it
					rpc.UnmapWindow(w.conn, w.boardWin)
				} else {
					// restore: show board again and repaint the bg to wipe the help
					rpc.MapWindow(w.conn, w.boardWin)
					rpc.ClearArea(w.conn, w.bgWin, 0, 0, 0, 0, false)
				}
				w.drawGame()
			} else {
				w.drawIntro() // drawIntro repaints the loader (wipes the help)
			}
			return true
		case 42: // g / G
			w.showGhost = !w.showGhost
			w.drawGame()
			return true
		case 54: // c / C - switch colour <-> mono theme
			w.toggleTheme()
			return true
		case 35, 86, 21: // + (German main/KP, US =/+)
			w.cycleRes(+1)
			return true
		case 61, 82, 20: // - (German main/KP, US -/_)
			w.cycleRes(-1)
			return true
		}

		// while the help page is open the game is paused: ignore everything else
		if w.showHelp {
			return true
		}

		if curState == stateIntro {
			curState = statePlaying
			paused = false
			gs = game.New()
			rpc.SetWindowBackgroundPixmap(w.conn, w.bgWin, w.bg)
			rpc.ClearArea(w.conn, w.bgWin, 0, 0, 0, 0, false)
			rpc.MapWindow(w.conn, w.boardWin)
			w.drawGame()
			return true
		}

		if e.Detail == 36 && gs.GameOver { // Return - restart after game over
			gs.Reset()
			paused = false
			w.drawGame()
			return true
		}

		if !gs.GameOver {
			switch e.Detail {
			case 65: // space - pause / resume
				paused = !paused
			case 113, 104: // left, h
				if !paused {
					gs.MoveLeft()
				}
			case 114, 108: // right, l
				if !paused {
					gs.MoveRight()
				}
			case 116, 106: // down, j
				if !paused {
					gs.MoveDown()
				}
			case 111, 107: // up, k
				if !paused {
					gs.Rotate()
				}
			case 36: // Return - hard drop
				if !paused {
					gs.HardDrop()
				}
			}
			w.drawGame()
		}
	}
	return true
}

// recreateWindows tears down and rebuilds the windows for the current
// resolution/theme, reloading the backgrounds and rescaling the glyphs.
func (w *TetrisWin) recreateWindows() {
	screen := w.conn.Setup.Screens[0]
	screenW := int(screen.Width)
	screenH := int(screen.Height)

	// destroy all 3 windows (children before parent)
	rpc.DestroyWindow(w.conn, w.boardWin)
	rpc.DestroyWindow(w.conn, w.bgWin)
	rpc.DestroyWindow(w.conn, w.frameWin.XID)
	w.conn.SendAndWait(&request.GetInputFocusRequest{}) // round-trip barrier

	// reset XIDs so Create() doesn't think they already exist
	w.frameWin.XID = 0
	w.bgWin = 0
	w.boardWin = 0

	w.createWin(screenW, screenH)

	switch curState {
	case stateIntro:
		w.drawIntro()
	case statePlaying:
		rpc.SetWindowBackgroundPixmap(w.conn, w.bgWin, w.bg)
		rpc.ClearArea(w.conn, w.bgWin, 0, 0, 0, 0, false)
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
	w.recreateWindows()
}

func cleanup() {
	sidplayer.Stop()
}

// ---- game loop ----

var doneCh = make(chan struct{})

func gameLoop(conn *proto_core.X11Conn, win *TetrisWin) {
	lastTick := time.Now()
	for {
		select {
		case <-doneCh:
			return
		default:
		}
		select {
		case ev := <-conn.Events():
			conn.DeliverWindowEvent(ev)
		case <-time.After(16 * time.Millisecond):
			if curState != statePlaying || gs.GameOver || paused || win.showHelp {
				continue
			}
			speed := time.Duration(gs.TickSpeed()) * time.Millisecond
			if time.Since(lastTick) >= speed {
				if !gs.MoveDown() {
					gs.LockPiece()
				}
				lastTick = time.Now()
				win.drawGame()
			}
		}
	}
}

func pickBestRes(screenW, screenH int) int {
	best := 0
	for i, r := range resolutions {
		if r.w <= screenW && r.h <= screenH {
			best = i
		}
	}
	return best
}

func main() {
	conn, err := proto.Dial("")
	errPanic(err, "connecting")
	defer conn.Close()
	defer cleanup()

	tkConn := tk_core.MakeTkConn(conn)

	screen := conn.Setup.Screens[0]
	screenW := int(screen.Width)
	screenH := int(screen.Height)

	win := &TetrisWin{
		conn:   conn,
		tkConn: &tkConn,
		resIdx: pickBestRes(screenW, screenH),
	}
	win.createWin(screenW, screenH)

	// auto-fullscreen if chosen resolution is near screen size (WM struts would clip it)
	if resolutions[win.resIdx].h*10 >= screenH*9 {
		win.toggleFullscreen()
	}

	gs = game.New()
	sidplayer.Start(sidData)

	curState = stateIntro
	gameLoop(conn, win)
}
