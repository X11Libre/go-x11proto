// Command terminal-aa is demo/terminal with antialiased TrueType text (see
// tk/font/ttf) instead of the X core bitmap font — otherwise identical: one
// top-level window, one Term widget, spawning $SHELL (or /bin/sh) on a real
// PTY. Same no-tabs, no-menu, single-process scope as demo/terminal, for the
// same reason (see its doc comment): tabbing belongs in a separate
// XEmbed-style program, not in the terminal widget itself.
//
// Usage: terminal-aa [shell-command]
package main

import (
	"log"
	"os"

	"github.com/X11Libre/go-x11proto/proto"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font/ttf"
	tk_render "github.com/X11Libre/go-x11proto/tk/render"
	"github.com/X11Libre/go-x11proto/tk/term"
)

// ttfPath is a system-installed monospace font; a real terminal would need
// embedding or fontconfig discovery instead of a hardcoded path (see
// demo/fonttest's history for the same caveat).
const ttfPath = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"

func main() {
	conn, err := proto.DialBE("")
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	tk := tk_core.MakeTkConn(conn)

	rdr, err := tk_render.Open(&tk)
	if err != nil {
		log.Fatalf("RENDER extension unavailable: %v", err)
	}
	face, err := ttf.Open(&tk, rdr, ttfPath, 13, 96)
	if err != nil {
		log.Fatalf("open TrueType face: %v", err)
	}
	defer face.Close()

	t := &term.Term{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: &tk},
			Parent:   tk.GetRoot(),
			Name:     "terminal-aa",
			X:        50,
			Y:        50,
			W:        800,
			H:        480,
			// Matches FgRGB/BgRGB below: without this, the X server paints
			// its own default (white) background for the window before the
			// first antialiased Draw() ever runs — a white flash on startup
			// and on every resize, even with double buffering fixing the
			// per-keypress repaint flicker.
			SetBackPixel: true,
			BackPixel:    conn.DefaultBlackPixel(),
		},
		AAFace:   face,
		AARender: rdr,
		FgRGB:    [3]byte{0xff, 0xff, 0xff},
		BgRGB:    [3]byte{0x00, 0x00, 0x00},
		Type:     term.XTerm256Color,
		OnTitle: func(s string) {
			// A real title update would call an X11 SetWMName-equivalent on
			// t.Window; not exposed by tk_core.Window as a runtime setter
			// yet, so this is logged for now rather than silently dropped.
			log.Printf("title: %s", s)
		},
		OnExit: func(err error) {
			os.Exit(0)
		},
	}
	if len(os.Args) > 1 {
		t.Shell = os.Args[1]
	}

	if err := t.Init(); err != nil {
		log.Fatalf("init: %v", err)
	}
	if err := t.Start(); err != nil {
		log.Fatalf("start shell: %v", err)
	}

	t.RunLoop(conn)
}
