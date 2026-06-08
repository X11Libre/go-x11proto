# go-x11proto

A pure Go X11 client protocol library (ALGPLv3+, work-in-progress).

## Module

- `github.com/X11Libre/go-x11proto` (Go 1.22, no external dependencies)

## Directory layout

| Path | Role |
|---|---|
| `proto/base/` | Low-level types, read/write buffers, base request primitives |
| `proto/core/` | Core X11 connection, event types, request encoding, opcodes, atoms |
| `proto/rpc/` | High-level RPC wrappers around raw requests (CreateWindow, MapWindow, etc.) |
| `tk/core/` | High-level toolkit: `TkConn`, `Window` with `WindowHandler` interface |
| `tk/widget/` | Widgets (`Button`) built on `tk/core` |
| `xts/` | Integration tests (require a running X server) |

## Commands

- `go build ./...` — build all packages
- `make test` — runs `go test -v github.com/X11Libre/go-x11proto/...`
- `make fmt` — runs `go fmt`
- `go build .` — builds the example binary in the module root (main package)

## Testing

- Tests in `xts/` are integration tests that connect to a live X display.
- Each test is run in both little-endian (`connectLE`) and big-endian (`connectBE`) mode.
- Event wait timeout is 2 seconds (`waitForEvent`).
- No mocking layer; tests require `$DISPLAY`.

## Architecture notes

- Two connection modes: `proto.Dial("")` for LE, `proto.DialBE("")` for BE. Passing `""` uses `$DISPLAY`.
- Low-level: use `core.NewConn(display, bigEndian)` directly, encode/decode with `base.ReadBuffer`/`base.WriteBuffer`.
- High-level: wrap via `tk_core.MakeTkConn(conn)`, use `Window` with a `WindowHandler` that implements `HandleWindowEvent(events.Event) bool`.
- Event loop: `conn.SimpleEventLoop()` blocks reading events and dispatches to registered window handlers.
- Window registration: `conn.RegisterWindowHandler(xid, w)` is called inside `tk_core.Window.Create()`.
- Only `"fixed"` font is used everywhere.
- RPC convenience layer (`proto/rpc/`) is the primary way to make requests in application code.

## Style conventions

- Package aliases: `proto_base`, `proto_core`, `proto_rpc`, `tk_core`, `tk_widget`.
- Event handler dispatch uses type switch on `events.Event`.
- GCs and fonts are lazily initialized (checked for `== 0`).
- init base.Rectangle via unkeyed fields is ok.
- always run `make fmt` when changing or creating .go files
