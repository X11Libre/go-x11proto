// Command editor is a small xedit-style text editor built entirely from the
// go-x11proto toolkit: a menu bar, an editable TextView with a Scrollbar, and a
// status line, laid out in a Frame. It demonstrates the tk/keyboard, tk/font,
// tk/clipboard and tk/widget editor pieces.
//
// Usage: editor [file]
package main

import (
	"log"
	"os"

	"github.com/X11Libre/go-x11proto/proto"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

func main() {
	conn, err := proto.DialBE("")
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	tk := tk_core.MakeTkConn(conn)

	var fname string
	if len(os.Args) > 1 {
		fname = os.Args[1]
	}

	ed := &Editor{conn: conn, tk: &tk}
	if err := ed.Init(fname); err != nil {
		log.Fatalf("init: %v", err)
	}

	conn.SimpleEventLoop()
}
