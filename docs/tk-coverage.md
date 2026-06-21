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

## Notes

- The text methods (`PutText8`, `ImageText8/16`, `PolyText16`) need an opened
  font and are not exercised live; their underlying requests have byte-level
  encode tests in `proto/core/request`.
- Subwindow / reparent / circulate Window methods are thin pass-throughs to rpc
  calls that are directly round-trip tested in `xts/window_ops_test.go`
  (`TestReparentAndSubwindows`).
