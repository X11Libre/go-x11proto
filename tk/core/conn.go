package core

import (
	"fmt"
	proto_base "github.com/X11Libre/go-x11proto/proto/base"
	proto_core "github.com/X11Libre/go-x11proto/proto/core"
	proto_rpc "github.com/X11Libre/go-x11proto/proto/rpc"
)

type TkConn struct {
	X11Conn    *proto_core.X11Conn
	RootWindow Window
	FontMap    map[string]proto_base.FONT
	AtomMap    map[string]proto_base.ATOM
}

func (tkc *TkConn) GetRoot() *Window {
	if tkc.RootWindow.Drawable.Invalid() {
		tkc.RootWindow.XID = tkc.X11Conn.DefaultRoot()
	}
	return &tkc.RootWindow
}

func (tkc *TkConn) GetFont(name string) (proto_base.FONT, error) {
	if tkc.FontMap == nil {
		tkc.FontMap = make(map[string]proto_base.FONT)
	}

	if fontId, ok := tkc.FontMap[name]; ok {
		return fontId, nil
	}

	fontId, err := proto_rpc.OpenFont(tkc.X11Conn, "fixed")
	if err != nil {
		return 0, fmt.Errorf("TkConn::GetFont(\"%s\"): %w", name, err)
	}

	tkc.FontMap[name] = fontId
	return fontId, nil
}

// InternAtom returns the atom for the given name, caching the result so each
// name is only interned once per connection.
func (tkc *TkConn) InternAtom(name string) (proto_base.ATOM, error) {
	if tkc.AtomMap == nil {
		tkc.AtomMap = make(map[string]proto_base.ATOM)
	}

	if atom, ok := tkc.AtomMap[name]; ok {
		return atom, nil
	}

	atom, err := proto_rpc.InternAtom(tkc.X11Conn, name)
	if err != nil {
		return 0, fmt.Errorf("TkConn::InternAtom(\"%s\"): %w", name, err)
	}

	tkc.AtomMap[name] = atom
	return atom, nil
}

func MakeTkConn(conn *proto_core.X11Conn) TkConn {
	return TkConn{X11Conn: conn}
}
