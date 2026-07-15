// Command terminal is a minimal terminal emulator built entirely from the
// go-x11proto toolkit's tk/term package: one top-level window, one Term
// widget, spawning $SHELL (or /bin/sh) on a real PTY. It demonstrates the
// package's intended minimum viable use — no tabs, no config file, no menu —
// deliberately left for a separate program to add (see tk/term's doc
// comment on why tabbing belongs outside this package entirely, XEmbed-style
// like suckless' tabbed, rather than built into the terminal itself).
//
// Usage: terminal [shell-command]
package main

import (
	"encoding/base64"
	"log"
	"os"

	"github.com/X11Libre/go-x11proto/proto"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font"
	"github.com/X11Libre/go-x11proto/tk/term"
)

func main() {
	conn, err := proto.DialBE("")
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	tk := tk_core.MakeTkConn(conn)

	f, err := font.Open(conn, "fixed")
	if err != nil {
		log.Fatalf("open font: %v", err)
	}
	defer f.Close(conn)

	t := &term.Term{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: &tk},
			Parent:   tk.GetRoot(),
			Name:     "terminal",
			X:        50,
			Y:        50,
			W:        800,
			H:        480,
		},
		Font: f,
		Type: term.XTerm256Color,
		OnTitle: func(s string) {
			// A real title update would call an X11 SetWMName-equivalent on
			// t.Window; not exposed by tk_core.Window as a runtime setter
			// yet, so this is logged for now rather than silently dropped.
			log.Printf("title: %s", s)
		},
		OnClipboard: func(sel, data string) {
			name := sel
			switch {
			case sel == "c":
				name = "CLIPBOARD"
			case sel == "p" || sel == "P" || sel == "0":
				name = "PRIMARY"
			case sel == "s" || sel == "S":
				name = "PRIMARY+CLIPBOARD"
			}
			if data == "?" {
				log.Printf("clipboard query: %s", name)
				return
			}
			if dec, err := base64.StdEncoding.DecodeString(data); err == nil {
				log.Printf("clipboard set: %s data=%q", name, string(dec))
			} else {
				log.Printf("clipboard set: %s (bad base64: %v)", name, err)
			}
		},
		OnHyperlink: func(params, uri string) {
			if uri == "" {
				log.Printf("hyperlink end")
			} else {
				log.Printf("hyperlink: params=%q uri=%s", params, uri)
			}
		},
		OnNotify: func(msg string) {
			log.Printf("notify: %s", msg)
		},
		OnOSC777: func(payload string) {
			log.Printf("osc777: %s", payload)
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
