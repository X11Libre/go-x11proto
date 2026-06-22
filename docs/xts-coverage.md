# XTS live test coverage

Round-trip tests in `xts/` run against a throwaway **Xvfb** server (started by
`TestMain`, see `xts/main_test.go`), so they exercise the real wire protocol and
server replies without touching the developer's session, and are CI-friendly.

Legend:
- **RT** — round-trip tested: a reply value is verified, not just the absence of an error.
- **OK** — exercised against the server successfully (no error).
- **—** — not yet covered; the *reason* column says why.

## Windows (opcodes 1–15)

| # | Request | Status | Test / reason |
|---|---------|--------|---------------|
| 1 | CreateWindow | RT | `TestWindowGeometryRoundTrip` (bg-pixel + event-mask), verified via GetGeometry |
| 2 | ChangeWindowAttributes | OK | `TestChangeWindowAttributes` |
| 3 | GetWindowAttributes | RT | class verified |
| 4 | DestroyWindow | OK | several |
| 5 | DestroySubwindows | OK | `TestMiscRequests` |
| 6 | ChangeSaveSet | OK | `TestReparentAndSubwindows` (insert+delete) |
| 7 | ReparentWindow | RT | parent verified via QueryTree |
| 8 | MapWindow | OK | several |
| 9 | MapSubwindows | OK | `TestReparentAndSubwindows` |
| 10 | UnmapWindow | OK | several |
| 11 | UnmapSubwindows | OK | `TestReparentAndSubwindows` |
| 12 | ConfigureWindow | RT | `TestConfigureWindow` (move+resize), geometry verified |
| 13 | CirculateWindow | OK | `TestReparentAndSubwindows` |
| 14 | GetGeometry | RT | geometry verified |
| 15 | QueryTree | RT | parent verified |

## Properties & selections (16–24)

| # | Request | Status | Test / reason |
|---|---------|--------|---------------|
| 16 | InternAtom | RT | `TestAtoms` incl. only-if-exists variation |
| 17 | GetAtomName | RT | name round-trip |
| 18 | ChangeProperty | OK | `TestProperties` (string + 32-bit) |
| 19 | DeleteProperty | OK | `TestProperties` |
| 20 | GetProperty | RT | value/format/len verified |
| 21 | ListProperties | RT | membership verified |
| 22 | SetSelectionOwner | OK | `TestSelections` |
| 23 | GetSelectionOwner | RT | owner verified |
| 24 | ConvertSelection | OK | `TestMiscRequests` (no owner → SelectionNotify None) |

## Grabs & events (25–37)

| # | Request | Status | Test / reason |
|---|---------|--------|---------------|
| 25 | SendEvent | OK | `TestMiscRequests` (synthetic ClientMessage to root) |
| 26 | GrabPointer | RT | status verified |
| 27 | UngrabPointer | OK | `TestGrabs` |
| 28 | GrabButton | OK | `TestGrabs` |
| 29 | UngrabButton | OK | `TestGrabs` |
| 30 | ChangeActivePointerGrab | OK | `TestMiscRequests` |
| 31 | GrabKeyboard | RT | status verified |
| 32 | UngrabKeyboard | OK | `TestGrabs` |
| 33 | GrabKey | OK | `TestGrabs` |
| 34 | UngrabKey | OK | `TestGrabs` |
| 35 | AllowEvents | OK | `TestGrabs` |
| 36 | GrabServer | OK | `TestGrabs` |
| 37 | UngrabServer | OK | `TestGrabs` |

## Pointer / focus / keyboard (38–44)

| # | Request | Status | Test / reason |
|---|---------|--------|---------------|
| 38 | QueryPointer | RT | root verified |
| 39 | GetMotionEvents | OK | `TestGetMotionEvents` |
| 40 | TranslateCoordinates | RT | translated coords verified |
| 41 | WarpPointer | OK | `TestFocusCoordsMisc` |
| 42 | SetInputFocus | OK | `TestFocusCoordsMisc` |
| 43 | GetInputFocus | OK | `TestMiscRequests` |
| 44 | QueryKeymap | RT | 32-byte keymap verified |

## Fonts (45–52)

| # | Request | Status | Test / reason |
|---|---------|--------|---------------|
| 45 | OpenFont | OK | `TestFontOpen` (skipped if no fonts) |
| 46 | CloseFont | OK | `TestFontOpen` (skipped if no fonts) |
| 47 | QueryFont | OK | `TestFontOpen` (skipped if no fonts) |
| 48 | QueryTextExtents | OK | `TestFontOpen` (skipped if no fonts) |
| 49 | ListFonts | OK | `TestFontListing` |
| 50 | ListFontsWithInfo | OK | `TestFontListing` (multi-reply path) |
| 51 | SetFontPath | — | mutates global font path |
| 52 | GetFontPath | OK | `TestFontListing` |

## Pixmaps & graphics contexts (53–60)

| # | Request | Status | Test / reason |
|---|---------|--------|---------------|
| 53 | CreatePixmap | OK | several |
| 54 | FreePixmap | OK | several |
| 55 | CreateGC | OK | several |
| 56 | ChangeGC | OK | `TestGCOps` |
| 57 | CopyGC | OK | `TestGCOps` |
| 58 | SetDashes | OK | `TestGCOps` |
| 59 | SetClipRectangles | OK | `TestGCOps` |
| 60 | FreeGC | OK | several |

## Drawing (61–77)

| # | Request | Status | Test / reason |
|---|---------|--------|---------------|
| 61 | ClearArea | OK | `TestClearArea` |
| 62 | CopyArea | OK | `TestCopyAreaAndPlane` |
| 63 | CopyPlane | OK | `TestCopyAreaAndPlane` |
| 64 | PolyPoint | OK | `TestDrawingPrimitives` |
| 65 | PolyLine | OK | `TestDrawingPrimitives` |
| 66 | PolySegment | OK | `TestDrawingPrimitives` |
| 67 | PolyRectangle | OK | `TestDrawingPrimitives` |
| 68 | PolyArc | OK | `TestDrawingPrimitives` |
| 69 | FillPoly | OK | `TestDrawingPrimitives` |
| 70 | PolyFillRectangle | OK | `TestDrawingPrimitives` |
| 71 | PolyFillArc | OK | `TestDrawingPrimitives` |
| 72 | PutImage | RT | `TestPutGetImageRoundTrip` |
| 73 | GetImage | RT | pixel data verified against PutImage |
| 74 | PolyText8 | — | needs a font; unit-tested in `proto/core/request` |
| 75 | PolyText16 | — | needs a font; unit-tested in `proto/core/request` |
| 76 | ImageText8 | OK | `TestTextAndGlyphCursor` (skipped if no font) |
| 77 | ImageText16 | — | needs a font; unit-tested in `proto/core/request` |

## Colormaps & colours (78–92)

| # | Request | Status | Test / reason |
|---|---------|--------|---------------|
| 78 | CreateColormap | OK | `TestColormapLifecycle` |
| 79 | FreeColormap | OK | `TestColormapLifecycle` |
| 80 | CopyColormapAndFree | OK | `TestMiscRequests` |
| 81 | InstallColormap | OK | `TestColormapLifecycle` |
| 82 | UninstallColormap | OK | `TestColormapLifecycle` |
| 83 | ListInstalledColormaps | RT | non-empty list verified |
| 84 | AllocColor | RT | `TestColors` |
| 85 | AllocNamedColor | — | needs the server colour database |
| 86 | AllocColorCells | — | needs a dynamic (non-TrueColor) visual |
| 87 | AllocColorPlanes | — | needs a dynamic visual |
| 88 | FreeColors | OK | `TestColors` |
| 89 | StoreColors | — | needs a read/write colormap |
| 90 | StoreNamedColor | — | needs a read/write colormap + colour DB |
| 91 | QueryColors | RT | `TestColors` |
| 92 | LookupColor | — | needs the server colour database |

## Cursors (93–96)

| # | Request | Status | Test / reason |
|---|---------|--------|---------------|
| 93 | CreateCursor | OK | `TestCursor` |
| 94 | CreateGlyphCursor | OK | `TestTextAndGlyphCursor` (skipped if no "cursor" font) |
| 95 | FreeCursor | OK | `TestCursor` |
| 96 | RecolorCursor | OK | `TestCursor` |

## Server / keyboard / pointer control & misc (97–127)

| # | Request | Status | Test / reason |
|---|---------|--------|---------------|
| 97 | QueryBestSize | OK | `TestPointerKeyboardQueries` |
| 98 | QueryExtension | OK | `TestExtensionsAndHosts` |
| 99 | ListExtensions | OK | `TestExtensionsAndHosts` |
| 100 | ChangeKeyboardMapping | — | mutates global keyboard state |
| 101 | GetKeyboardMapping | RT | keysyms-per-keycode verified |
| 102 | ChangeKeyboardControl | — | mutates global state |
| 103 | GetKeyboardControl | OK | `TestPointerKeyboardQueries` |
| 104 | Bell | OK | `TestFocusCoordsMisc` |
| 105 | ChangePointerControl | OK | `TestServerControlRequests` |
| 106 | GetPointerControl | OK | `TestPointerKeyboardQueries` |
| 107 | SetScreenSaver | OK | `TestServerControlRequests` |
| 108 | GetScreenSaver | OK | `TestPointerKeyboardQueries` |
| 109 | ChangeHosts | OK | `TestServerControlRequests` (insert + delete) |
| 110 | ListHosts | OK | `TestExtensionsAndHosts` |
| 111 | SetAccessControl | OK | `TestServerControlRequests` (enable + restore) |
| 112 | SetCloseDownMode | OK | `TestServerControlRequests` |
| 113 | KillClient | — | destroys clients |
| 114 | RotateProperties | OK | `TestProperties` |
| 115 | ForceScreenSaver | OK | `TestServerControlRequests` |
| 116 | SetPointerMapping | — | mutates global state |
| 117 | GetPointerMapping | OK | `TestPointerKeyboardQueries` |
| 118 | SetModifierMapping | — | mutates global state |
| 119 | GetModifierMapping | RT | keycodes-per-modifier verified |
| 127 | NoOperation | OK | `TestFocusCoordsMisc` |

## Error-condition coverage (XTS parity)

Xorg's XTS (`xts5/Xproto`) devotes most of its per-request tests to error
conformance. `TestProtocolErrors` reproduces the **semantic** error checks by
driving a minimal bad request through `X11Conn.CheckRequest` (which reports the
server's `*core.RequestError`, including its `errorcode`) and asserting the
exact code:

| X error | provoked by |
|---------|-------------|
| BadRequest | request with an unassigned major opcode (200) |
| BadValue | GetImage with an invalid `format` |
| BadWindow | GetWindowAttributes on an unallocated id |
| BadPixmap | FreePixmap on an unallocated id |
| BadAtom | GetAtomName(0) |
| BadCursor | FreeCursor on an unallocated id |
| BadFont | CloseFont on a non-font id |
| BadMatch | GetImage rectangle exceeding the drawable |
| BadDrawable | GetGeometry on an unallocated id |
| BadColor | FreeColormap on an unallocated id |
| BadGC | FreeGC on an unallocated id |
| BadIDChoice | CreatePixmap reusing an in-use id |

Not reproduced: **BadLength** (XTS sends truncated requests at the wire level;
the high-level API always emits correct lengths, so it is neither reproducible
nor meaningful here), and **BadAccess / BadAlloc / BadName** (hard to trigger
deterministically / environment-dependent — e.g. dynamic-visual colormap cells,
the RGB colour database).

## Notes

- The remaining **—** rows are: dynamic-visual colour ops (cells/planes/store,
  needing a non-static visual Xvfb lacks), named colours (RGB database), the
  remaining keyboard/pointer mapping setters (ChangeKeyboardMapping,
  Set{Pointer,Modifier}Mapping, ChangeKeyboardControl, SetFontPath — need
  carefully-constructed valid arguments) and the client-destroying KillClient.
  All requests — including the untested-live ones — have byte-level
  encode/decode unit tests in `proto/core/request`.
- Request set: XTS's `xts5/Xproto` covers the same ~120 core requests; the live
  happy-path + the error table above bring this suite to broad parity with it,
  excluding the wire-level BadLength purposes noted above.
