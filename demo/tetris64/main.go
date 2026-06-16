package main

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	tetris_font "github.com/X11Libre/go-x11proto/demo/tetris64/font"
	"github.com/X11Libre/go-x11proto/demo/tetris64/game"
	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/base"
	proto_core "github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/xpm"
)

//go:embed assets/tetris_c64_frame_320_color.png
var frame320PNG []byte

//go:embed assets/tetris_c64_loader_320_color.png
var loader320PNG []byte

//go:embed assets/tetris.sid
var sidData []byte

type appState int

const (
	stateIntro appState = iota
	statePlaying
)

var curState = stateIntro
var gs *game.State

type resOpt struct {
	w, h  int
	scale int
	name  string
}

var resolutions = []resOpt{
	{320, 200, 1, "320"},
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
	levelX, levelY       int
	gameOverX, gameOverY int
}

var layouts = []resLayout{
	{cell: 8, bx: 120, by: 16, nx: 208, ny: 112, scoreX: 245, scoreY: 13, linesX: 245, linesY: 22, levelX: 248, levelY: 40, gameOverX: 120, gameOverY: 88},
	{cell: 42, bx: 755, by: 99, nx: 1248, ny: 605, scoreX: 1470, scoreY: 70, linesX: 1470, linesY: 119, levelX: 1488, levelY: 216, gameOverX: 720, gameOverY: 475},
	{cell: 55, bx: 1010, by: 143, nx: 1664, ny: 806, scoreX: 1960, scoreY: 93, linesX: 1960, linesY: 158, levelX: 1984, levelY: 288, gameOverX: 960, gameOverY: 633},
	{cell: 83, bx: 1513, by: 212, nx: 2496, ny: 1209, scoreX: 2940, scoreY: 140, linesX: 2940, linesY: 237, levelX: 2976, levelY: 432, gameOverX: 1440, gameOverY: 950},
}

type TetrisWin struct {
	conn   *proto_core.X11Conn
	tkConn *tk_core.TkConn

	frameWin tk_core.Window // top-level resizable frame (black bg)
	bgWin    base.WINDOW    // child of frame, holds bg pixmap, centered
	boardWin base.WINDOW    // child of bgWin, holds board rendering

	gcText  base.GC
	gcBlack  base.GC
	gcColors map[uint8]base.GC
	gcGhost  map[uint8]base.GC

	bg       base.PIXMAP
	loaderBg base.PIXMAP

	digitPix [10]base.PIXMAP // greyscale digit glyphs, rebuilt per resolution

	layout     resLayout
	scale      int
	resIdx     int
	fullscreen bool
	showHelp   bool
	showGhost  bool
	frameW     int
	frameH     int
}

var sidProc *os.Process

func errPanic(e error, s string) {
	if e != nil {
		panic(s + ": " + e.Error())
	}
}

// ---- raw request helper for requests without existing structs ----

type destroyWinReq struct {
	Window base.WINDOW
}

func (r *destroyWinReq) WriteInto(w *base.RequestWriter) error {
	w.SetOpcode(opcode.DestroyWindow)
	w.WriteCARD32(base.CARD32(r.Window))
	return nil
}

type getInputFocusReq struct{}

func (r *getInputFocusReq) WriteInto(w *base.RequestWriter) error {
	w.SetOpcode(opcode.GetInputFocus)
	return nil
}

type sendEventReq struct {
	propagate bool
	dest      base.WINDOW
	eventMask base.CARD32
	eventData [32]byte
}

func (r *sendEventReq) WriteInto(w *base.RequestWriter) error {
	w.SetOpcode(opcode.SendEvent)
	if r.propagate {
		w.SetParam0(1)
	}
	w.WriteCARD32(base.CARD32(r.dest))
	w.WriteCARD32(r.eventMask)
	w.WriteBytes(r.eventData[:])
	return nil
}

type configureWinReq struct {
	Window base.WINDOW
	X, Y   base.INT16
	setPos bool
}

func (r *configureWinReq) WriteInto(w *base.RequestWriter) error {
	w.SetOpcode(opcode.ConfigureWindow)
	var mask base.CARD16
	if r.setPos {
		mask |= 0x0003 // CW_X | CW_Y
	}
	w.WriteCARD32(base.CARD32(r.Window))
	w.WriteCARD16(mask)
	w.WriteCARD16(0) // padding to 4-byte alignment
	if r.setPos {
		w.WriteCARD32(base.CARD32(uint16(r.X)))
		w.WriteCARD32(base.CARD32(uint16(r.Y)))
	}
	return nil
}

// ---- asset helpers ----

func assetPathFor(name string) string {
	var candidates []string
	// Walk up from the working directory so the on-disk assets are found
	// whether the program is launched from the repo root, from demo/tetris64,
	// or any directory in between (lets edited PNGs take effect without a
	// rebuild). Fall back to assets next to the executable.
	if cwd, err := os.Getwd(); err == nil {
		d := cwd
		for i := 0; i < 8; i++ {
			candidates = append(candidates,
				filepath.Join(d, "demo/tetris64/assets", name),
				filepath.Join(d, "assets", name))
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "assets", name))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func loadFrame(resName string) []byte {
	if p := assetPathFor("tetris_c64_frame_" + resName + "_color.png"); p != "" {
		if d, err := os.ReadFile(p); err == nil {
			return d
		}
	}
	return frame320PNG
}

func loadLoader(resName string) []byte {
	if p := assetPathFor("tetris_c64_loader_" + resName + "_color.png"); p != "" {
		if d, err := os.ReadFile(p); err == nil {
			return d
		}
	}
	return loader320PNG
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

	w.bg = w.uploadBg(loadFrame(res.name))
	w.loaderBg = w.uploadBg(loadLoader(res.name))
	l := w.layout

	// top-level frame window (black background, resizable)
	w.frameWin = tk_core.Window{
		Drawable: tk_core.Drawable{
			Conn: w.tkConn,
		},
		Parent:    w.tkConn.GetRoot(),
		Name:      "C64 TETRIS",
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

	// bg window: child of frame, exactly image size, centered
	bgXID := base.WINDOW(w.conn.NextResourceID())
	_, err := w.conn.Send(&request.CreateWindowRequest{
		Depth:     0,
		Wid:       bgXID,
		Parent:    w.frameWin.XID,
		X:         0,
		Y:         0,
		Width:     base.CARD16(res.w),
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

	w.conn.Send(&sendEventReq{
		propagate: false,
		dest:      w.conn.DefaultRoot(),
		eventMask: 0xFFFFFF,
		eventData: ev,
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
}

func (w *TetrisWin) drawGame() {
	if w.gcBlack.Invalid() {
		gcid, err := rpc.CreateGC1(w.conn, 0x000000, 0x000000, 0)
		errPanic(err, "CreateGC1 black")
		w.gcBlack = gcid
	}
	if w.gcText.Invalid() {
		// washed-out light gray (C64 light-gray, #BABABA) of the original
		// score/lines/level readout — not stark white, to match the artwork.
		gcid, err := rpc.CreateGC1(w.conn, 0xBABABA, 0x000000, 0)
		errPanic(err, "CreateGC1 text")
		w.gcText = gcid
	}

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

	// clear score/lines/level text areas
	w.fillRects(w.bgWin, w.gcBlack, []base.Rectangle{
		{X: base.INT16(l.scoreX), Y: base.INT16(l.scoreY), Width: base.CARD16(48 * fs), Height: base.CARD16(8 * fs)},
		{X: base.INT16(l.linesX), Y: base.INT16(l.linesY), Width: base.CARD16(24 * fs), Height: base.CARD16(8 * fs)},
		{X: base.INT16(l.levelX), Y: base.INT16(l.levelY), Width: base.CARD16(16 * fs), Height: base.CARD16(8 * fs)},
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
	}

	// ghost piece (relative to boardWin)
	dist := gs.DropDistance()
	if w.showGhost && dist > 0 {
		p := gs.Current
		_, ghost := w.gcCol(uint8(p.Type))
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
				rects = append(rects, base.Rectangle{
					X: base.INT16(bx * l.cell), Y: base.INT16(by * l.cell),
					Width: base.CARD16(l.cell), Height: base.CARD16(l.cell),
				})
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

	bgDraw := tk_core.Drawable{Conn: w.tkConn, XID: w.bgWin}

	// score/lines/level use the greyscale digit pixmaps (scaled from masters)
	w.drawNumber(w.bgWin, base.INT16(l.scoreX), base.INT16(l.scoreY), fmt.Sprintf("%06d", gs.Score))
	w.drawNumber(w.bgWin, base.INT16(l.linesX), base.INT16(l.linesY), fmt.Sprintf("%03d", gs.Lines))
	w.drawNumber(w.bgWin, base.INT16(l.levelX), base.INT16(l.levelY), fmt.Sprintf("%02d", gs.Level))

	if gs.GameOver {
		tetris_font.DrawString(bgDraw, w.gcText,
			base.INT16(l.gameOverX), base.INT16(l.gameOverY), fs, "GAME OVER")
	}

	if w.showHelp {
		res := resolutions[w.resIdx]
		hx := 40 * res.w / 320
		hy := 24 * res.h / 200
		hbw := res.w - 2*hx
		hbh := res.h - 2*hy

		w.fillRects(w.bgWin, w.gcBlack, []base.Rectangle{
			{X: base.INT16(hx + 1), Y: base.INT16(hy + 1), Width: base.CARD16(hbw - 2), Height: base.CARD16(hbh - 2)},
		})
		w.fillRects(w.bgWin, w.gcText, []base.Rectangle{
			{X: base.INT16(hx), Y: base.INT16(hy), Width: base.CARD16(hbw), Height: 1},
			{X: base.INT16(hx), Y: base.INT16(hy + hbh), Width: base.CARD16(hbw), Height: 1},
			{X: base.INT16(hx), Y: base.INT16(hy), Width: 1, Height: base.CARD16(hbh)},
			{X: base.INT16(hx + hbw), Y: base.INT16(hy), Width: 1, Height: base.CARD16(hbh)},
		})

		titleX := (res.w - 4*8*fs) / 2
		tetris_font.DrawString(bgDraw, w.gcText,
			base.INT16(titleX), base.INT16(hy+12*res.h/200), fs, "HELP")
		sepY := hy + 20*res.h/200
		w.fillRects(w.bgWin, w.gcText, []base.Rectangle{
			{X: base.INT16(hx + 16), Y: base.INT16(sepY), Width: base.CARD16(hbw - 32), Height: 1},
		})

		helpLines := []struct {
			y    int
			k, a string
		}{
			{32, "ARROWS/WASD  Move", ""},
			{40, "DOWN / J      Soft drop", ""},
			{48, "UP / K        Rotate", ""},
			{56, "SPACE         Hard drop", ""},
			{64, "+ / -         Resolution", ""},
			{72, "F             Fullscreen", ""},
			{80, "G             Toggle ghost", ""},
			{88, "Q             Quit", ""},
		}
		for _, hl := range helpLines {
			yy := hy + hl.y*res.h/200
			tetris_font.DrawString(bgDraw, w.gcText,
				base.INT16(hx+16*res.w/320), base.INT16(yy), fs, hl.k)
		}

		closeX := (res.w - 20*8*fs) / 2
		tetris_font.DrawString(bgDraw, w.gcText,
			base.INT16(closeX), base.INT16(hy+112*res.h/200), fs, "PRESS F1 TO CLOSE")
	}
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

			// re-center bgWin
			res := resolutions[w.resIdx]
			bgX := (nw - res.w) / 2
			bgY := (nh - res.h) / 2
			if bgX < 0 {
				bgX = 0
			}
			if bgY < 0 {
				bgY = 0
			}
			w.conn.Send(&configureWinReq{
				Window: w.bgWin,
				X:      base.INT16(bgX),
				Y:      base.INT16(bgY),
				setPos: true,
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
		case 67: // F1
			w.showHelp = !w.showHelp
			w.drawGame()
			return true
		case 42: // g / G
			w.showGhost = !w.showGhost
			w.drawGame()
			return true
		case 35, 86, 21: // + (German main/KP, US =/+)
			w.cycleRes(+1)
			return true
		case 61, 82, 20: // - (German main/KP, US -/_)
			w.cycleRes(-1)
			return true
		}

		if curState == stateIntro {
			curState = statePlaying
			gs = game.New()
			rpc.SetWindowBackgroundPixmap(w.conn, w.bgWin, w.bg)
			rpc.ClearArea(w.conn, w.bgWin, 0, 0, 0, 0, false)
			rpc.MapWindow(w.conn, w.boardWin)
			w.drawGame()
			return true
		}

		if e.Detail == 36 { // Return - restart
			gs.Reset()
			w.drawGame()
			return true
		}

		if !gs.GameOver {
			switch e.Detail {
			case 113, 104: // left, h
				gs.MoveLeft()
			case 114, 108: // right, l
				gs.MoveRight()
			case 116, 106: // down, j
				gs.MoveDown()
			case 111, 107: // up, k
				gs.Rotate()
			case 65: // space
				gs.HardDrop()
			}
			w.drawGame()
		}
	}
	return true
}

func (w *TetrisWin) cycleRes(dir int) {
	n := len(resolutions)
	w.resIdx = (w.resIdx + dir + n) % n

	screen := w.conn.Setup.Screens[0]
	screenW := int(screen.Width)
	screenH := int(screen.Height)

	// destroy all 3 windows (children before parent)
	w.conn.Send(&destroyWinReq{Window: w.boardWin})
	w.conn.Send(&destroyWinReq{Window: w.bgWin})
	w.conn.Send(&destroyWinReq{Window: w.frameWin.XID})
	w.conn.SendAndWait(&getInputFocusReq{})

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

// ---- SID music ----

func startMusic() {
	sidPath, err := exec.LookPath("sidplayfp")
	if err != nil {
		return
	}
	tmpFile := filepath.Join(os.TempDir(), "go-x11proto-tetris.sid")
	if err := os.WriteFile(tmpFile, sidData, 0644); err != nil {
		return
	}
	cmd := exec.Command(sidPath, "-t0", tmpFile)
	cmd.Stdin, _ = os.Open(os.DevNull)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return
	}
	sidProc = cmd.Process
}

func stopMusic() {
	if sidProc != nil {
		sidProc.Signal(os.Kill)
		sidProc.Wait()
		sidProc = nil
	}
}

func cleanup() {
	stopMusic()
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
			if curState != statePlaying || gs.GameOver {
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
	startMusic()

	curState = stateIntro
	gameLoop(conn, win)
}
