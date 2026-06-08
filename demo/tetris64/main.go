package main

import (
	"time"

	"github.com/X11Libre/go-x11proto/demo/tetris64/sidplayer"
	"github.com/X11Libre/go-x11proto/proto"
	proto_core "github.com/X11Libre/go-x11proto/proto/core"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

// theme selects the asset set; it is the background asset subdir
// ("color" or "mono"). Currently fixed. (Asset loading lives in assets.go.)
var theme = "color"

type appState int

const (
	stateIntro appState = iota
	statePlaying
)

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

func errPanic(e error, s string) {
	if e != nil {
		panic(s + ": " + e.Error())
	}
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
			if win.state != statePlaying || win.gs.GameOver || win.paused || win.showHelp {
				continue
			}
			speed := time.Duration(win.gs.TickSpeed()) * time.Millisecond
			if time.Since(lastTick) >= speed {
				if !win.gs.MoveDown() {
					win.gs.LockPiece()
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

	tkConn := tk_core.MakeTkConn(conn)

	screen := conn.Setup.Screens[0]
	screenW := int(screen.Width)
	screenH := int(screen.Height)

	win := &TetrisWin{
		conn:   conn,
		tkConn: &tkConn,
		resIdx: pickBestRes(screenW, screenH),
		music:  sidplayer.New(),
	}
	defer win.music.Stop()
	win.createWin(screenW, screenH)

	// auto-fullscreen if chosen resolution is near screen size (WM struts would clip it)
	if resolutions[win.resIdx].h*10 >= screenH*9 {
		win.toggleFullscreen()
	}

	win.music.Start(sidData)

	// win starts in the intro state (the zero value); startGame creates the
	// game state on the first key press.
	gameLoop(conn, win)
}
