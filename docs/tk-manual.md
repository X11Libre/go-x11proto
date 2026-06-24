# tk — a small X11 toolkit for go-x11proto

`tk/` is a lightweight widget toolkit built directly on the `proto` wire layer.
It wraps the raw X11 protocol in Go objects — connections, windows, drawables,
graphics contexts — and adds fonts, keyboard decoding, a clipboard, theming and
a handful of widgets (labels, buttons, menus, an editable text view, a
scrollbar, a layout frame).

It links no C libraries: everything is the Go protocol client talking to the
server. This manual is the programming reference; for the exhaustive
request/widget test matrix see [tk-coverage.md](tk-coverage.md), and for runnable
code see `demo/simple`, `demo/editor` and `demo/tetris64`.

## Contents

1. [Layers and packages](#1-layers-and-packages)
2. [Getting started](#2-getting-started)
3. [The connection: `TkConn`](#3-the-connection-tkconn)
4. [Drawables and drawing](#4-drawables-and-drawing)
5. [Graphics contexts: `GC`](#5-graphics-contexts-gc)
6. [Pixmaps](#6-pixmaps)
7. [Windows and the event loop](#7-windows-and-the-event-loop)
8. [Fonts and text: `tk/font`](#8-fonts-and-text-tkfont)
9. [Keyboard input: `tk/keyboard`](#9-keyboard-input-tkkeyboard)
10. [Widgets: `tk/widget`](#10-widgets-tkwidget)
11. [Clipboard and selections: `tk/clipboard`](#11-clipboard-and-selections-tkclipboard)
12. [Theming: `tk/theme` and `tk/xsettings`](#12-theming-tktheme-and-tkxsettings)
13. [Images: `tk/xpm`](#13-images-tkxpm)
14. [Compositing: `tk/render`](#14-compositing-tkrender)
15. [Putting it together](#15-putting-it-together)
16. [Conventions and gotchas](#16-conventions-and-gotchas)

---

## 1. Layers and packages

```
proto/            raw wire protocol (base types, requests, rpc, events, ext/*)
  proto/core         X11Conn, the connection + event channel
  proto/base         wire scalar types: CARD8/16/32, INT16, WINDOW, GC, ATOM …
tk/core           TkConn, Window, Drawable, GC, Pixmap
tk/font           server fonts + metrics (a TextRenderer)
tk/keyboard       keycode → keysym → rune
tk/widget         Label, Button, Menu, MenuBar, TextView, Scrollbar, Frame
tk/clipboard      PRIMARY/CLIPBOARD text transfer
tk/theme          desktop font DPI / family from XSETTINGS
tk/xsettings      the XSETTINGS protocol (client + manager)
tk/xpm            XPM / PNG / image.Image → uploadable pixmaps
tk/render         the RENDER extension (compositing, solid fills)
```

A recurring detail: some helpers take the **raw** connection `*proto/core.X11Conn`
(`font.Open`, `keyboard.Load`, `theme.Load`, `clipboard.New`), while the drawing
layer hangs off `*tk/core.TkConn`. A `TkConn` exposes the raw one as its
`X11Conn` field, so you can always get from one to the other.

Wire scalar types come from `proto/base`: positions are `base.INT16`, sizes are
`base.CARD16`, pixels/masks are `base.CARD32`, resource ids are `base.WINDOW` /
`base.GC` / `base.PIXMAP` / `base.FONT` / `base.ATOM`.

---

## 2. Getting started

A minimal program: connect, make a `TkConn`, create and map a window, run the
event loop.

```go
package main

import (
	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

type Hello struct {
	tk_core.Window
}

func (h *Hello) HandleWindowEvent(ev events.Event) bool {
	// repaint on Expose, etc.
	return true
}

func main() {
	conn, err := proto.DialBE("") // "" = $DISPLAY; DialBE/Dial choose byte order
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	tk := tk_core.MakeTkConn(conn)

	win := &Hello{Window: tk_core.Window{
		Drawable:  tk_core.Drawable{Conn: &tk},
		Parent:    tk.GetRoot(),
		Name:      "hello",
		X:         100, Y: 100, W: 400, H: 300,
		EventMask: 0xFFFFFF, // all events; usually you pick specific masks
	}}
	win.SetWindowHandler(win)
	_ = win.Create()
	_ = win.Map()

	conn.SimpleEventLoop()
}
```

`proto.Dial` opens a little-endian client connection, `proto.DialBE` a
big-endian one; both speak to the same server. `MakeTkConn` returns a value, so
take its address (`&tk`) for the `Drawable.Conn` field.

---

## 3. The connection: `TkConn`

```go
type TkConn struct {
	X11Conn *proto_core.X11Conn // the underlying wire connection
	// …atom cache, default-font cache…
}

func MakeTkConn(conn *core.X11Conn) TkConn

func (c *TkConn) GetRoot() *Window                          // the default-screen root
func (c *TkConn) GetFont(name string) (base.FONT, error)    // OpenFont, cached
func (c *TkConn) InternAtom(name string) (base.ATOM, error) // InternAtom, cached
func (c *TkConn) CreateGC1(fg, bg base.CARD32, font base.FONT) (*GC, error)
func (c *TkConn) CreatePixmap(depth base.CARD8, ref base.DRAWABLE, w, h base.CARD16) (*Pixmap, error)
```

`InternAtom` and `GetFont` cache by name, so calling them repeatedly is cheap.
Black and white pixel values come from the raw connection:
`tk.X11Conn.DefaultBlackPixel()`, `DefaultWhitePixel()`, `DefaultRoot()`.

---

## 4. Drawables and drawing

`Drawable` is the common base of `Window` and `Pixmap` — anything you can draw
on. It carries a `*TkConn` and a resource id, and offers the drawing requests:

```go
type Drawable struct {
	Conn *TkConn
	XID  base.DRAWABLE
}

func (d Drawable) FillRect(gc base.GC, x, y base.INT16, w, h base.CARD16) error
func (d Drawable) FillRects(gc base.GC, rects []base.Rectangle) error
func (d Drawable) PolyLine(gc base.GC, coordMode base.CARD8, pts []base.Point) error
func (d Drawable) PolySegment(gc base.GC, segs []base.Segment) error
func (d Drawable) PolyRectangle(gc base.GC, rects []base.Rectangle) error
func (d Drawable) PolyArc(gc base.GC, arcs []base.Arc) error
func (d Drawable) PolyPoint(gc base.GC, coordMode base.CARD8, pts []base.Point) error
func (d Drawable) FillPoly(gc base.GC, shape, coordMode base.CARD8, pts []base.Point) error
func (d Drawable) PolyFillArc(gc base.GC, arcs []base.Arc) error
func (d Drawable) PutText8(gc base.GC, x, y base.INT16, text string) error   // fg only
func (d Drawable) ImageText8(gc base.GC, x, y base.INT16, text string) error // fg+bg cell
func (d Drawable) PutImage(gc base.GC, format, depth base.CARD8, w, h base.CARD16, data []byte) error
func (d Drawable) GetImage(format base.CARD8, x, y base.INT16, w, h base.CARD16, mask base.CARD32) (*request.GetImageReply, error)
func (d Drawable) CopyArea(dst base.DRAWABLE, gc base.GC, sx, sy, dx, dy base.INT16, w, h base.CARD16) error
func (d Drawable) CopyPlane(dst base.DRAWABLE, gc base.GC, sx, sy, dx, dy base.INT16, w, h base.CARD16, plane base.CARD32) error
func (d Drawable) GetGeometry() (*request.GetGeometryReply, error)
func (d Drawable) Invalid() bool // true if the id is unset/zero
```

Text drawn with `PutText8` paints only the foreground (good for overlays on an
existing background); `ImageText8` also fills the glyph cells with the GC's
background colour (good for cleanly overwriting old text). Both take a baseline
`y`. The `tk/font` package wraps these with metrics and top-left positioning
(§8).

---

## 5. Graphics contexts: `GC`

A `GC` bundles the drawing parameters (colours, font, line width, …) the server
applies to each request.

```go
gc, err := tk.CreateGC1(black, white, font) // foreground, background, font (0 = none)
defer gc.Free()

gc.SetForeground(0xff0000)
gc.SetBackground(white)
gc.SetFont(fontID)
gc.Change(&request.ChangeGCRequest{ValueMask: request.GC_MASK_LINE_WIDTH, LineWidth: 2})
```

Drawing requests take the `base.GC` id, i.e. `gc.XID`:

```go
win.FillRect(gc.XID, 10, 10, 50, 50)
```

---

## 6. Pixmaps

A `Pixmap` is an off-screen drawable (same drawing methods, via the embedded
`Drawable`). Use it for double-buffering, icons, or as a window background.

```go
pm, _ := tk.CreatePixmap(screenDepth, root, 64, 64)
defer pm.Free()
pm.FillRect(gc.XID, 0, 0, 64, 64)
pm.CopyArea(win.XID, gc.XID, 0, 0, 0, 0, 64, 64) // blit to a window
```

`tk/xpm` builds ready-to-use pixmaps from images (§13).

---

## 7. Windows and the event loop

### The `Window` struct

Fill the fields before calling `Create`:

```go
type Window struct {
	Drawable               // Conn + XID
	Parent    *Window      // nil → root
	ParentXID base.WINDOW  // alternative to Parent
	Name      string       // WM_NAME, set on Create when non-empty
	X, Y      base.INT16
	W, H      base.CARD16
	EventMask base.CARD32  // see proto/core/events/event_mask

	BorderWidth    base.CARD16
	BackPixel      base.CARD32 // honored when SetBackPixel
	SetBackPixel   bool
	BorderPixel    base.CARD32 // honored when SetBorderPixel
	SetBorderPixel bool

	WinHandler WindowHandler
}
```

Lifecycle and manipulation:

```go
func (w *Window) Create() error
func (w Window)  Map() / Unmap() / Destroy() error
func (w Window)  Move(x, y) / Resize(w, h) / MoveResize(x, y, w, h) error
func (w Window)  Raise() / Lower() error
func (w Window)  ClearArea(x, y, w, h base.CARD16, exposures bool) error
func (w Window)  SetName(string) error
func (w Window)  SetBackgroundPixmap(base.PIXMAP) error // tk_core.ParentRelative for "transparent"
func (w Window)  SetOverrideRedirect(bool) error        // popups the WM should ignore
func (w Window)  Reparent / MapSubwindows / CirculateUp / … // less common
func (w Window)  GetGeometry / GetAttributes / QueryTree    // round-trip queries
```

`Create` registers the window with the connection so its events are routed to
its handler, and sets a white background unless you set `SetBackPixel`.

### Handlers and event routing

A window delegates events to a `WindowHandler`:

```go
type WindowHandler interface {
	HandleWindowEvent(ev events.Event) bool
}
```

Call `w.SetWindowHandler(self)` (usually the widget embeds `Window` and is its
own handler). The connection dispatches each event to the handler **of the
window the event is for** — it keys on `ev.ReceiverWindow()`. So independent
widgets each handle only their own events; you rarely write a central switch.

Type-switch on the concrete event:

```go
func (w *MyWin) HandleWindowEvent(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		w.redraw()
	case *events.ButtonPressEvent:
		fmt.Println("button", e.Key, "at", e.EventX, e.EventY)
	case *events.ConfigureEvent:
		w.W, w.H = e.Width, e.Height
	}
	return true
}
```

### The loop

```go
conn.SimpleEventLoop()            // for ev := range conn.Events() { conn.DeliverWindowEvent(ev) }
```

`SimpleEventLoop` reads the event channel and dispatches forever. For custom
loops use `conn.Events()` (a `<-chan events.Event`), dispatch with
`conn.DeliverWindowEvent(ev)`, and register non-window handlers (e.g. a
clipboard) with `conn.RegisterWindowHandler(win, handler)`.

To receive a class of events the window's `EventMask` must request it — e.g.
`event_mask.Exposure` to get `ExposeEvent`, `KeyPress`, `ButtonPress`,
`StructureNotify` (for `ConfigureEvent`), etc.

---

## 8. Fonts and text: `tk/font`

A `Font` wraps a server font with the metrics needed to lay text out, and is a
ready `TextRenderer` (§10).

```go
f, err := font.Open(conn.X11Conn, "fixed") // or a full XLFD
defer f.Close(conn.X11Conn)

f.Height()                 // ascent + descent (line height)
f.TextWidth("hello")       // pixel width
f.RuneWidth('w')           // one glyph's advance
f.IndexAtX("hello", px)    // char index nearest a pixel offset (caret hit-testing)

gc, _ := tk.CreateGC1(black, white, f.ID)
f.DrawText(win.Drawable, gc.XID, x, y, 0, "hello")   // top-left positioned, fg only
f.DrawTextBG(win.Drawable, gc.XID, x, y, "hello")    // fills the cell background too
```

`font.Open` opens by name and queries metrics; `font.Query(conn, id)` builds the
metrics for a font you already opened (e.g. from `theme.OpenFont`). The `scale`
argument of `DrawText`/`Measure` exists for the `TextRenderer` interface and is
ignored by core fonts.

---

## 9. Keyboard input: `tk/keyboard`

A `KeyPressEvent` carries a raw keycode (`e.Key`) and modifier state
(`e.State`). `tk/keyboard` turns that into a keysym, a Unicode rune and a
logical key, applying the Shift/CapsLock rules.

```go
km, _ := keyboard.Load(conn.X11Conn) // snapshot the server mapping once

// inside HandleWindowEvent, for *events.KeyPressEvent e:
k := km.Lookup(e.Key, e.State)
switch k.Key {
case keyboard.KeyEnter:    // …
case keyboard.KeyBackspace:// …
case keyboard.KeyLeft, keyboard.KeyRight, keyboard.KeyUp, keyboard.KeyDown:
case keyboard.KeyNone:
	if k.Printable() {     // a normal character, no Ctrl/Alt
		insert(k.Rune)
	}
}
```

`keyboard.Event` fields: `Keysym uint32`, `Rune rune` (0 if not printable),
`Key Key` (`KeyNone` for ordinary characters; otherwise `KeyEnter`,
`KeyBackspace`, `KeyDelete`, `KeyTab`, `KeyEscape`, the arrows, `KeyHome`,
`KeyEnd`, `KeyPageUp`, `KeyPageDown`), and the modifier booleans `Shift`,
`Ctrl`, `Alt`. `Printable()` is true for an insertable character (has a rune, no
Ctrl/Alt, not a special key).

`TextView` already does all of this; you only touch `tk/keyboard` directly for a
custom key-handling widget.

---

## 10. Widgets: `tk/widget`

### The widget pattern

Every widget embeds `tk_core.Window`. You fill the embedded window's geometry
(`Parent`, `X/Y/W/H`) and the widget's own fields, then call `Init`. `Init`
selects the event masks it needs, creates and maps the window, and installs the
widget as its own handler. The caller owns any `GC` it passes in.

### `Label`

A single line of text via a pluggable `TextRenderer`.

```go
lbl := &tk_widget.Label{
	Window:   tk_core.Window{Drawable: tk_core.Drawable{Conn: &tk}, Parent: &win, X: 0, Y: 0, W: 200, H: 22,
		EventMask: base.CARD32(event_mask.Exposure)},
	Renderer: f,          // e.g. a *font.Font
	Gc:       gc.XID,
	Align:    tk_widget.AlignLeft, // AlignCenter (default) | AlignLeft | AlignRight
	Text:     "status",
}
lbl.Init()
lbl.SetText("updated")    // repaints
```

Set `Transparent: true` to give the label a `ParentRelative` background, so only
the glyphs are painted over whatever is behind it (used for the tetris "GAME
OVER" overlay).

### `Button`

```go
btn := &tk_widget.Button{
	Window:        tk_core.Window{Drawable: tk_core.Drawable{Conn: &tk}, Parent: &win,
		Name: "OK", X: 10, Y: 10, W: 80, H: 30, EventMask: 0xFFFFFF},
	OnButtonPress: func() { /* clicked */ },
}
btn.Init()
```

### `Menu` and `MenuBar`

`MenuItem` describes one entry: a leaf (`Label` + `OnClick`), a cascade
(non-nil `Submenu`), or a `Separator`. An optional `Accel` string is shown
right-aligned and wired as a hotkey.

```go
type MenuItem struct {
	Label     string
	OnClick   func()
	Submenu   []MenuItem
	Separator bool
	Accel     string // e.g. "Ctrl+O", "Ctrl+Shift+S"
}
```

A **menu bar** across the top of a window:

```go
bar := &tk_widget.MenuBar{Window: tk_core.Window{
	Drawable: tk_core.Drawable{Conn: &tk}, Parent: &win, X: 0, Y: 0, W: 500}}
bar.AddMenu("File", []tk_widget.MenuItem{
	{Label: "Open", Accel: "Ctrl+O", OnClick: openFile},
	{Label: "Save", Accel: "Ctrl+S", OnClick: saveFile},
	{Separator: true},
	{Label: "Quit", Accel: "Ctrl+Q", OnClick: func() { os.Exit(0) }},
})
bar.Init()
```

A standalone **context menu** (pop up on right-click at root coordinates):

```go
ctx := &tk_widget.Menu{Items: items}
ctx.Drawable.Conn = &tk
ctx.Init()
// in a ButtonPress handler, e.Key == 3:
ctx.Popup(base.INT16(e.RootX), base.INT16(e.RootY))
```

**Accelerators.** If an item has an `Accel` that parses (modifiers
`Ctrl`/`Shift`/`Alt` plus a single key), `bar.HandleKey(k)` / `menu.HandleKey(k)`
will fire its `OnClick` for a matching `keyboard.Event` and return true. Forward
your focused widget's keys to it — e.g. `textview.OnKey = bar.HandleKey` — and
every hotkey is just the menu's own accelerator, with no duplicated key table.

### `TextView`

A multi-line, editable text area: a line buffer drawn with a font, a caret,
keyboard editing/navigation, mouse selection, and vertical scrolling.

```go
tv := &tk_widget.TextView{
	Window: tk_core.Window{Drawable: tk_core.Drawable{Conn: &tk}, Parent: &win,
		X: 0, Y: 22, W: 484, H: 456},
	Font: f,
}
tv.Init()             // also loads the keyboard map
tv.SetText("hello\nworld")
s := tv.Text()        // buffer as one string
```

Content / cursor / scrolling:

```go
tv.LineCount(); tv.VisibleLines(); tv.TopLine()
tv.ScrollTo(line)
tv.SelectedText(); tv.DeleteSelection(); tv.Insert("pasted text")
tv.Focus()            // grab the keyboard
```

Hooks let it drive the rest of a UI:

```go
tv.OnChange = func()        { markDirty(); refreshStatus() }
tv.OnScroll = func()        { scrollbar.SetRange(tv.LineCount(), tv.VisibleLines(), tv.TopLine()) }
tv.OnSelect = func(s string){ primary.Own(s) }    // finished mouse selection → own PRIMARY
tv.OnKey    = bar.HandleKey                         // accelerators; return true = handled
```

Other fields: `ReadOnly bool`, `Fg`/`Bg base.CARD32`, `SelectionBg base.CARD32`.
`TextView` tracks its own size on `ConfigureNotify`, so a `Frame` can resize it.

### `Scrollbar`

A vertical track with a draggable thumb, bound to a line range.

```go
sb := &tk_widget.Scrollbar{Window: tk_core.Window{
	Drawable: tk_core.Drawable{Conn: &tk}, Parent: &win, X: 484, Y: 22, W: 16, H: 456}}
sb.OnScroll = func(top int) { tv.ScrollTo(top) }
sb.Init()
sb.SetRange(tv.LineCount(), tv.VisibleLines(), tv.TopLine())
```

Pair it with the text view by forwarding `tv.OnScroll → sb.SetRange` and
`sb.OnScroll → tv.ScrollTo`.

### `Frame`

A border-layout container that re-lays its children on resize: optional
`Top`/`Bottom` bars span the full width, `Left`/`Right` take the height between
them, and `Center` fills the rest.

```go
frame := &tk_widget.Frame{Window: tk_core.Window{
	Drawable: tk_core.Drawable{Conn: &tk}, Name: "app", X: 100, Y: 100, W: 700, H: 500}}
frame.Init()                                    // create children parented to &frame.Window…
frame.Top    = &tk_widget.Slot{Win: &bar.Window,    Extent: 22}
frame.Bottom = &tk_widget.Slot{Win: &status.Window, Extent: 22}
frame.Right  = &tk_widget.Slot{Win: &sb.Window,     Extent: 16}
frame.Center = &tv.Window
frame.Relayout(700, 500)
```

A `Slot` is `{Win *tk_core.Window; Extent int}` (the height of a top/bottom bar
or width of a left/right bar). `Frame` re-runs the layout automatically on
`ConfigureNotify`; `OnLayout` fires after each pass.

### `TextRenderer`

The interface that decouples text-drawing widgets (`Label`, `TextView`) from any
particular font implementation:

```go
type TextRenderer interface {
	DrawText(d tk_core.Drawable, gc base.GC, x, y base.INT16, scale int, s string) error
	Measure(scale int, s string) (w, h int)
}
```

`*font.Font` satisfies it (core server fonts). The tetris demo provides a bitmap
glyph renderer; you can supply your own (scalable glyphs, RENDER, …).

---

## 11. Clipboard and selections: `tk/clipboard`

Text copy/paste over the X selections. A `Clipboard` owns a small window and
handles both roles — owning a selection and requesting another owner's text.

```go
win, _ := rpc.CreateWindow1(conn, conn.DefaultRoot(), -10, -10, 1, 1, clipboard.EventMask)
cb, _ := clipboard.New(conn, win, "CLIPBOARD") // or "PRIMARY"
conn.RegisterWindowHandler(win, cb)            // so it serves requests in the loop

// copy:
cb.Own(selectedText)

// paste (asynchronous; result arrives via the event loop):
cb.OnPaste = func(s string) { textview.Insert(s) }
cb.RequestText()
```

`Own(text)` takes the selection and serves `UTF8_STRING`/`STRING`/`TARGETS`
requests; `RequestText()` asks the current owner to convert, delivering the
result to `OnPaste` from `HandleX11WindowEvent`. A synchronous
`GetText(timeout)` exists for tools that run outside an event loop (it reads the
event channel directly). `Owned()` reports current ownership.

---

## 12. Theming: `tk/theme` and `tk/xsettings`

`tk/theme` reads the desktop's font hints (via XSETTINGS — the channel GTK/Qt
use) and turns them into concrete sizes, so your text scales with the running
desktop instead of a fixed pixel size.

```go
th := theme.Load(conn)              // falls back to 96 dpi / "Sans 10"
px := th.FontPixelSize()            // UI font height in pixels at the theme DPI
f, _ := th.OpenFont(conn)           // a core font at that pixel size ("fixed" fallback)
th.PointsToPixels(12.0)             // points → pixels at the theme DPI
```

`tk/xsettings` is the protocol underneath: `xsettings.NewClient(conn, screen)`
with `Get()`/`Watch(onChange)` reads the live settings (`DPI()`, `FontName()`,
`ThemeName()`, …); `xsettings.NewManager(conn, screen)` lets you *publish*
settings (own `_XSETTINGS_S<n>` and set the property). Most apps only need
`tk/theme`.

---

## 13. Images: `tk/xpm`

Decode an image and upload it as a pixmap you can use as a window background or
blit with `CopyArea`.

```go
img, _ := xpm.DecodeBytes(xpmBytes)       // also: xpm.Decode(r), xpm.DecodePNG, xpm.FromImage(image.Image)
pm, _ := img.Upload(conn, conn.DefaultRoot())
win.SetBackgroundPixmap(pm)               // visible on next expose/map
```

`Image` is `{Width, Height int; Data []byte /* RGBA */}`. `FromImage` accepts any
`image.Image`, so you can bridge from Go's standard image codecs.

---

## 14. Compositing: `tk/render`

A tk wrapper over the RENDER extension for alpha compositing and solid fills.

```go
rdr, err := render.Open(&tk)          // QueryExtension + version
argb, _ := rdr.ARGB32()               // a standard picture format
pic, _ := rdr.PictureFor(pixmap.Drawable, argb, render.PictureValues{})
pic.FillRect(render.PictOpSrc, render.Color{Alpha: 0xffff}, 0, 0, 32, 32)
pic.Composite(render.PictOpOver, src, nil, 0, 0, 0, 0, dx, dy, w, h)
pic.Free()
```

Use it when you need translucency or antialiased compositing that core drawing
can't express.

---

## 15. Putting it together

`demo/editor` is the canonical worked example — it composes almost the whole
toolkit into an xedit-style editor. The skeleton:

```go
tk := tk_core.MakeTkConn(conn)
th := theme.Load(conn)
f  := openMonospace(conn, th.FontPixelSize())   // tk/font

frame := &tk_widget.Frame{ /* toplevel */ }
frame.Init()

bar := &tk_widget.MenuBar{ /* parent &frame.Window */ }   // File/Edit with accelerators
bar.AddMenu("File", []tk_widget.MenuItem{ {Label:"Open", Accel:"Ctrl+O", OnClick: open}, … })
bar.Init()

tv := &tk_widget.TextView{ /* parent &frame.Window */, Font: f }; tv.Init()
sb := &tk_widget.Scrollbar{ /* parent &frame.Window */ };        sb.Init()
status := &tk_widget.Label{ /* AlignLeft */ };                   status.Init()

frame.Top, frame.Bottom = &tk_widget.Slot{Win:&bar.Window,Extent:22}, &tk_widget.Slot{Win:&status.Window,Extent:22}
frame.Right, frame.Center = &tk_widget.Slot{Win:&sb.Window,Extent:16}, &tv.Window
frame.Relayout(700, 500)

tv.OnScroll = func(){ sb.SetRange(tv.LineCount(), tv.VisibleLines(), tv.TopLine()) }
sb.OnScroll = func(top int){ tv.ScrollTo(top) }
tv.OnKey    = bar.HandleKey            // menu accelerators drive every hotkey

clip, _ := clipboard.New(conn, clipWin, "CLIPBOARD")   // copy/paste
conn.RegisterWindowHandler(clipWin, clip)

conn.SimpleEventLoop()
```

See `demo/simple` for a smaller tour (background pixmap, menu bar, context menu,
a child window and a button) and `demo/tetris64` for custom drawing with a
bitmap-font `TextRenderer`.

---

## 16. Conventions and gotchas

- **Two connection types.** Drawing hangs off `*tk/core.TkConn`; `font.Open`,
  `keyboard.Load`, `theme.Load`, `clipboard.New` take the raw
  `*proto/core.X11Conn`. Reach it as `tk.X11Conn` (or keep the original `conn`).
  `render.Open` takes the `*TkConn`.
- **Set `EventMask` for what you want.** No `Exposure` → no repaint events; no
  `StructureNotify` → no `ConfigureEvent` on resize, etc. Widgets OR-in what they
  need in `Init`, but a hand-built window must request masks itself.
- **Fill fields before `Create`/`Init`.** Geometry, `Parent`, `Name`, colours
  and `EventMask` are read at creation time.
- **Parent children correctly.** A child's `Parent` must point at its container
  window (`&frame.Window`), and the parent must exist (be `Create`d) first.
- **Coordinates are typed.** `base.INT16` for positions, `base.CARD16` for
  sizes, `base.CARD32` for pixels/masks — convert explicitly.
- **Focus.** Keyboard events go to the focused window. `TextView` grabs focus on
  click; call `tv.Focus()` to set it programmatically (e.g. after a dialog).
- **Resource cleanup.** `GC.Free`, `Pixmap.Free`, `Window.Destroy`,
  `font.Close` release server resources; long-lived widgets typically keep
  theirs for their lifetime.
- **Byte order.** `Dial` (LE) and `DialBE` (BE) both work against any server;
  the wire codecs handle swapping. The test suite exercises both.
- **Errors.** Protocol errors surface as `*proto/core.RequestError` (with
  `Code`, `MajorOpcode`, …); check with `errors.As`.

For the per-request and per-widget test matrix, see
[tk-coverage.md](tk-coverage.md).
