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
| `SetIcon` / `SetIconRGBA` | ChangeProperty (_NET_WM_ICON) | RT | `TestTkSetIcon` (property layout verified) |
| `EnableWMDelete` / `IsWMDelete` | ChangeProperty (WM_PROTOCOLS) | RT | `TestTkWMDelete`, `TestIsWMDelete` |

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
| `Label` | single line of text via a pluggable `TextRenderer`; Align left/center/right; optional ParentRelative ("transparent") background | OK | `TestTkLabel`, `TestLabelAlignX` (offline) |
| `Menu` | override-redirect popup with separators and cascading submenus; press-drag-release selection driven by one pointer grab on the top menu | OK | `TestTkMenu`, `TestTkContextMenu` (separators + 3-layer submenus) |
| `MenuBar` | horizontal title strip; a press pops up the title's `Menu` below it | OK | `TestTkMenu` |
| `Button` | bordered push button: centred label (tk/font), inverted while pressed, OnButtonPress | OK | exercised by the simple demo and `dialog.Confirm` (`TestTkConfirm`) |
| `TextView` | multi-line editable text: caret, insert/delete/nav, Tab (expanded), Shift+nav & mouse selection, undo/redo, find/replace, vertical+horizontal & wheel scroll; OnChange/OnScroll/OnSelect/OnKey hooks | OK | `TestTextView*`, `TestSearchFrom`, `TestReplaceAll`, `TestFindNext` (offline), `TestTkTextView`, `TestTkWheelScroll` (live) |
| `Scrollbar` | vertical track + thumb sized to the line range; click-to-page, drag-to-scroll; OnScroll | OK | `TestThumbGeom*` (offline), `TestTkScrollbar` (live, bound to a TextView) |
| `Frame` | border layout (Top/Bottom/Left/Right/Center), re-lays children on resize | OK | `TestBorderLayout*` (offline), `TestTkFrame` (live geometry) |

## Keyboard (tk/keyboard)

Turns raw keycodes into keysyms/runes for text input.

| API | Role | Status | Test |
|-----|------|--------|------|
| `Load` | snapshot the server keyboard mapping (GetKeyboardMapping over the setup range) | RT | `TestKeyboardMap` (live) |
| `Map.Lookup` | keycode+state -> keysym, rune, logical Key; Shift/CapsLock case rules | OK | `TestLookup*` (offline), `TestKeyboardMap` (live 'a'/'A'/Enter) |

## Font (tk/font)

Core server font wrapped with metrics; a concrete `TextRenderer`.

| API | Wraps | Status | Test |
|-----|-------|--------|------|
| `Open` / `Query` | OpenFont + QueryFont (metrics) | RT | `TestTkFont` |
| `Height` / `TextWidth` / `RuneWidth` / `IndexAtX` | glyph advances for layout & caret hit-testing | OK | `TestTkFont` |
| `DrawText` / `DrawTextBG` | PutText8 / ImageText8 (top-left positioned) | OK | `TestTkFont` (glyphs verified via GetImage) |

## Clipboard (tk/clipboard)

Text copy/paste over the PRIMARY/CLIPBOARD selections.

| API | Wraps | Status | Test |
|-----|-------|--------|------|
| `Own` / `Serve` | SetSelectionOwner + answer SelectionRequest (UTF8_STRING/STRING/TARGETS) | OK | `TestTkClipboard` (live, two connections) |
| `RequestText` / `GetText` | ConvertSelection + read the result property | RT | `TestTkClipboard`, `TestTkClipboardNoOwner` |

## XSETTINGS (tk/xsettings)

Property-based, server-mediated desktop settings (font DPI, font/theme name,
colours) - the channel GTK/Qt use.

| API | Role | Status | Test |
|-----|------|--------|------|
| `Client` (`Get` / `Watch` / `ManagerWindow` / accessors `DPI`/`FontName`/`ThemeName`/…) | read the published settings + live-watch changes | RT | `TestXSettings`, `TestXSettingsWatch` |
| `Manager` (`NewManager` / `Set` / `Close`) | own `_XSETTINGS_S<n>` (real timestamp) + publish via the `_XSETTINGS_SETTINGS` property | OK | `TestXSettings` (live round-trip) |
| codec (`encode`/`decode`) | the binary settings format, both byte orders | OK | `TestCodecRoundTrip` (offline) |

## Theme (tk/theme)

Turns the XSETTINGS hints into concrete sizes so a toolkit can scale with the
desktop.

| API | Role | Status | Test |
|-----|------|--------|------|
| `Load` | read `Xft/DPI` + `Gtk/FontName` (defaults 96 / "Sans 10") | RT | `TestTheme` (publishes 192dpi/12pt, verified) |
| `PointsToPixels` / `FontPixelSize` | point→pixel at the theme DPI | OK | `TestPointsToPixels`, `TestTheme` |
| `OpenFont` | core font at the themed pixel size, "fixed" fallback | OK | `TestTheme` |
| `parseFontName` | split "Family … Npt" | OK | `TestParseFontName` (offline) |

## Dialogs (tk/dialog)

Reusable dialog windows built on the toolkit.

| API | Role | Status | Test |
|-----|------|--------|------|
| `FilePicker` (`Init` / `Open` / `Draw` / `CurrentDir`) | file-open chooser: listing, selection, scroll, keyboard + mouse navigation; `Floating` option | OK | `TestTkFilePicker`, `TestTkFilePickerFloating` (live), `TestArrange*`/`TestTarget`/`TestClampSel`/`TestScrollTop`/`TestReadDirLive` (offline) |
| `Confirm` (`Init` / `Draw`) | yes/no dialog: two Button widgets + keyboard (Enter/y, Esc/n); `Floating` option | OK | `TestTkConfirm` (live, both callbacks) |

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
