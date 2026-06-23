# tk Drawable/Window operation coverage

The `tk/core` layer wraps the raw `proto/rpc` calls in convenient methods on
`Drawable` (any drawable: window or pixmap) and `Window`. This matrix lists each
method, the rpc/request it wraps, and the live test (`xts/tk_test.go`, run
against the Xvfb harness) that exercises it.

Legend:
- **RT** — round-trip tested: a reply value is verified.
- **OK** — exercised against the server (no error).
- **rpc** — the underlying rpc is covered by an `xts` test, the tk wrapper is a thin pass-through.

## Drawable methods

| Method | Wraps | Status | Test |
|--------|-------|--------|------|
| `FillRect` / `FillRects` | PolyFillRectangle | OK | `TestTkWindowOps` |
| `PutText8` | PolyText8 | — | needs a font |
| `PutImage` | PutImage | rpc | `TestPutGetImageRoundTrip` |
| `CopyArea` | CopyArea | OK | `TestTkDrawableOps` |
| `CopyPlane` | CopyPlane | OK | `TestTkDrawableOps` |
| `PolyPoint` | PolyPoint | OK | `TestTkDrawableOps` |
| `PolyLine` | PolyLine | OK | `TestTkWindowOps` |
| `PolySegment` | PolySegment | OK | `TestTkDrawableOps` |
| `PolyRectangle` | PolyRectangle | OK | `TestTkDrawableOps` |
| `PolyArc` | PolyArc | OK | `TestTkDrawableOps` |
| `FillPoly` | FillPoly | OK | `TestTkDrawableOps` |
| `PolyFillArc` | PolyFillArc | OK | `TestTkDrawableOps` |
| `ImageText8` | ImageText8 | — | needs a font |
| `ImageText16` | ImageText16 | — | needs a font |
| `PolyText16` | PolyText16 | — | needs a font |
| `GetGeometry` | GetGeometry | RT | `TestTkDrawableOps` / `TestTkWindowOps` (dimensions verified) |
| `GetImage` | GetImage | rpc | `TestPutGetImageRoundTrip` |

## Window methods

| Method | Wraps | Status | Test |
|--------|-------|--------|------|
| `Create` | CreateWindow | OK | `TestTkWindowOps` |
| `Map` | MapWindow | OK | `TestTkWindowOps` |
| `Unmap` | UnmapWindow | OK | `TestTkWindowOps` |
| `SetName` | ChangeProperty (WM_NAME) | rpc | exercised by the tetris demo; `TestProperties` |
| `Destroy` | DestroyWindow | OK | `TestTkWindowOps` |
| `ClearArea` | ClearArea | OK | `TestTkWindowOps` |
| `MapSubwindows` | MapSubwindows | rpc | `TestReparentAndSubwindows` |
| `UnmapSubwindows` | UnmapSubwindows | rpc | `TestReparentAndSubwindows` |
| `Reparent` | ReparentWindow | rpc | `TestReparentAndSubwindows` |
| `CirculateUp` / `CirculateDown` | CirculateWindow | rpc | `TestReparentAndSubwindows` |
| `GetAttributes` | GetWindowAttributes | RT | `TestTkWindowOps` (class verified) |
| `QueryTree` | QueryTree | RT | `TestTkWindowOps` (parent verified) |
| `ChangeAttributes` | ChangeWindowAttributes | OK | `TestTkWindowOps` |
| `Configure` | ConfigureWindow | OK | `TestTkWindowOps` (via Move/Resize/MoveResize) |
| `Move` | ConfigureWindow | RT | `TestTkWindowOps` (geometry verified) |
| `Resize` | ConfigureWindow | RT | `TestTkWindowOps` (geometry verified) |
| `MoveResize` | ConfigureWindow | OK | `TestTkWindowOps` |
| `Raise` | ConfigureWindow (stack=Above) | OK | `TestTkWindowOps` |
| `Lower` | ConfigureWindow (stack=Below) | OK | `TestTkWindowOps` |
| `SetBackgroundPixmap` | ChangeWindowAttributes (background-pixmap) | OK | `TestTkSetBackgroundPixmap` |

## Connection / resource helpers

| Method | Wraps | Status | Test |
|--------|-------|--------|------|
| `TkConn.InternAtom` | InternAtom (cached) | RT | `TestTkInternAtom` (resolves + caches, verified via GetAtomName) |
| `TkConn.CreateGC1` | CreateGC | OK | `TestTkGC` |
| `GC.SetForeground` / `SetBackground` / `Change` | ChangeGC | OK | `TestTkGC` |
| `GC.Free` | FreeGC | OK | `TestTkGC` |
| `TkConn.CreatePixmap` | CreatePixmap | OK | `TestTkPixmap` / `TestTkGC` |
| `Pixmap.Free` | FreePixmap | OK | `TestTkPixmap` |
| `Pixmap` (drawing / copy / query) | *embeds Drawable* | RT | `TestTkPixmap` (GetGeometry verified) |

## Widgets (tk/widget)

| Widget | Notes | Status | Test |
|--------|-------|--------|------|
| `Label` | single line of centred text via a pluggable `TextRenderer`; optional ParentRelative ("transparent") background | OK | `TestTkLabel` (Init / Draw / SetText / transparent) |
| `Menu` | override-redirect popup with separators and cascading submenus; press-drag-release selection driven by one pointer grab on the top menu | OK | `TestTkMenu`, `TestTkContextMenu` (separators + 3-layer submenus) |
| `MenuBar` | horizontal title strip; a press pops up the title's `Menu` below it | OK | `TestTkMenu` |

## XSETTINGS (tk/xsettings)

Property-based, server-mediated desktop settings (font DPI, font/theme name,
colours) - the channel GTK/Qt use.

| API | Role | Status | Test |
|-----|------|--------|------|
| `Client` (`Get` / `ManagerWindow` / accessors `DPI`/`FontName`/`ThemeName`/…) | read the published settings | RT | `TestXSettings` (values verified) |
| `Manager` (`NewManager` / `Set` / `Close`) | own `_XSETTINGS_S<n>` + publish via the `_XSETTINGS_SETTINGS` property | OK | `TestXSettings` (live round-trip) |
| codec (`encode`/`decode`) | the binary settings format, both byte orders | OK | `TestCodecRoundTrip` (offline) |

## RENDER (tk/render)

tk-layer wrapper over `proto/ext/render` (the RENDER extension).

| API | Wraps | Status | Test |
|-----|-------|--------|------|
| `Open` / `Version` | QueryExtension + QueryVersion | OK | `TestTkRender` |
| `Formats` / `StandardFormat` / `ARGB32` / `RGB24` | QueryPictFormats (+ cache) | RT | `TestTkRender` (ARGB32 found) |
| `NewPicture` / `PictureFor` | CreatePicture | OK | `TestTkRender` |
| `Picture.Fill` / `FillRect` | FillRectangles | RT | `TestTkRender` (fill verified via GetImage) |
| `Picture.Composite` | Composite | OK | `TestTkRender` |
| `Picture.Change` | ChangePicture | OK | `TestTkRender` |
| `Picture.Free` | FreePicture | OK | `TestTkRender` |

## Notes

- The text methods (`PutText8`, `ImageText8/16`, `PolyText16`) need an opened
  font and are not exercised live; their underlying requests have byte-level
  encode tests in `proto/core/request`.
- Subwindow / reparent / circulate Window methods are thin pass-throughs to rpc
  calls that are directly round-trip tested in `xts/window_ops_test.go`
  (`TestReparentAndSubwindows`).
