package proto

import (
	"github.com/X11Libre/go-x11proto/proto/core"
)

// Dial connects to an X11 display using little-endian byte order.
// Empty display_name reads $DISPLAY.
func Dial(display_name string) (*core.X11Conn, error) {
	return core.NewConn(display_name, false)
}

// DialBE connects to an X11 display using big-endian byte order.
func DialBE(display_name string) (*core.X11Conn, error) {
	return core.NewConn(display_name, true)
}

// DialAuth connects to an X11 display using explicit auth credentials.
//
// Parameters:
//   - display_name: X11 display (e.g. ":0"); empty reads $DISPLAY
//   - authorityPath: path to an Xauthority file; empty reads XAUTHORITY or ~/.Xauthority
//   - authProto: auth protocol name (e.g. "MIT-MAGIC-COOKIE-1")
//   - authData: raw auth token bytes
func DialAuth(display_name, authorityPath, authProto string, authData []byte) (*core.X11Conn, error) {
	var protoBytes []byte
	if authProto != "" {
		protoBytes = []byte(authProto)
	}
	return core.NewConnWithAuth(display_name, false, authorityPath, protoBytes, authData)
}

// DialAuthBE connects to an X11 display in big-endian mode with explicit auth.
func DialAuthBE(display_name, authorityPath, authProto string, authData []byte) (*core.X11Conn, error) {
	var protoBytes []byte
	if authProto != "" {
		protoBytes = []byte(authProto)
	}
	return core.NewConnWithAuth(display_name, true, authorityPath, protoBytes, authData)
}
