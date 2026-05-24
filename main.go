package main

import (
	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"log"
)

func errPanic(e error, s string) {
	if e != nil {
		log.Printf("Error: %s\n", s)
		panic(e)
	}
}

func main() {
	conn, err := proto.DialBE("")
	errPanic(err, "connecting")
	defer conn.Close()

	exts, err := rpc.ListExtensions(conn)
	for _, v := range exts {
		ei, _ := rpc.QueryExtension(conn, v)
		log.Printf("EXTENSION: %+v\n", ei)
	}

	tkConn := tk_core.MakeTkConn(conn)

	win := MyWindow{
		Window: tk_core.Window{
			Parent:    tkConn.GetRoot(),
			Conn:      &tkConn,
			Name:      "HELLO WORLD EXAMPLE",
			X:         50,
			Y:         50,
			W:         500,
			H:         600,
			EventMask: 0xFFFFFF,
		},
	}
	win.Init()

	conn.SimpleEventLoop()
}
