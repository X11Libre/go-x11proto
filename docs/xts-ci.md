# Running XTS against an xserver build (CI integration)

`xts/` is a live X11 **wire-protocol** test suite: each test opens a real
connection, issues real requests and checks the server's real replies/errors
(see [xts-coverage.md](xts-coverage.md) for the per-request matrix). Because it
talks the protocol rather than linking any client library, it can be pointed at
**any** X server built from the xserver tree (Xvfb, Xephyr, Xorg, Xwayland).

The goal is to run this suite as part of the **XLibre xserver** build/CI, against
the just-built server — eventually as a replacement for Xorg's aging XTS.

## How it finds the server

`xts/main_test.go` launches a throwaway server on a private display (via
`-displayfd`, read back from stdout), so the tests never touch an existing
session. The launch is environment-configurable:

| Variable | Default | Meaning |
|----------|---------|---------|
| `XTS_XSERVER` | `Xvfb` | server executable: a `$PATH` name or an absolute/relative path |
| `XTS_XSERVER_ARGS` | `-screen 0 1280x1024x24` | extra args (besides `-displayfd 1`, added automatically) |

If the executable can't be started it falls back to `$DISPLAY`.

## Local / one-shot

```sh
# against a freshly built Xvfb from an xserver build tree:
contrib/xts/run-xts.sh /path/to/xserver/build/hw/vfb/Xvfb

# against any server + custom args:
contrib/xts/run-xts.sh /path/to/build/hw/kdrive/ephyr/Xephyr -screen 1280x1024x24

# defaults to the Xvfb on $PATH:
contrib/xts/run-xts.sh
```

`run-xts.sh` just sets `XTS_XSERVER`/`XTS_XSERVER_ARGS` and runs `go test ./xts/...`
from the module root; `GOTESTFLAGS` overrides the `go test` flags.

### Convenience wrappers

For everyday local runs there are thin wrappers around `run-xts.sh` (all accept
extra args to override the default screen, and pass `GOTESTFLAGS` through):

| Script | Server | Isolation |
|--------|--------|-----------|
| `contrib/xts/run-xvfb.sh` | spawns Xvfb (headless) | full — touches no session |
| `contrib/xts/run-xephyr.sh` | spawns Xephyr (nested window on `$DISPLAY`) | tests hit the nested server, not yours |
| `contrib/xts/run-xnest.sh` | spawns Xnest (nested window on `$DISPLAY`) | tests hit the nested server, not yours |
| `contrib/xts/run-local.sh` | the server already on `$DISPLAY` (no spawn) | **none — destructive** |

`run-xvfb.sh` is the default choice. `run-xephyr.sh` / `run-xnest.sh` need a
parent `$DISPLAY` to show their window but still isolate the tests in the nested
server. `run-local.sh` runs the **destructive** suite directly against
`$DISPLAY` (it grabs the server, warps the pointer, changes pointer/screensaver/
access-control settings), so it warns and prompts for confirmation — pass
`--yes` or set `XTS_YES=1` to skip the prompt. It works via `XTS_XSERVER=none`,
which tells the harness to use the existing `$DISPLAY` instead of spawning.

## GitHub Actions (in the xserver repo)

Add a step to the job that already builds the server — same checkout, no artifact
plumbing needed:

```yaml
      # ... after the xserver meson build, with the binaries in ./build ...
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - name: Check out go-x11proto (XTS)
        uses: actions/checkout@v4
        with:
          repository: X11Libre/go-x11proto
          path: go-x11proto
      - name: Run XTS against the built Xvfb
        run: ./go-x11proto/contrib/xts/run-xts.sh "$GITHUB_WORKSPACE/build/hw/vfb/Xvfb"
```

`run-xts.sh` exits non-zero if any test fails, so the job fails as usual. For
machine-readable results, install `gotest.tools/gotestsum` and set
`GO="gotestsum --junitfile xts.xml --"` (or wrap `go test` accordingly).

## Meson test integration (in the xserver repo)

The suite can also be wired in as a meson test so `meson test` runs it. In the
relevant `meson.build`:

```meson
go = find_program('go', required: false)
if go.found()
  test('xts',
    files('contrib/xts/run-xts.sh'),       # vendored / submoduled go-x11proto
    args: [Xvfb.full_path()],              # the build's Xvfb target
    env: {'GOFLAGS': '-mod=mod'},
    timeout: 300,
    is_parallel: false,
  )
endif
```

(Adjust the path to `run-xts.sh` and the `Xvfb` target reference to match how
go-x11proto is made available in the build — git submodule, subproject, or a CI
checkout.)

## Notes

- Needs a Go toolchain (the module targets **go 1.22**) and the built server
  binary. No X client libraries are required.
- The suite is read-mostly and self-contained: it creates its own windows,
  pixmaps, GCs, colormaps, etc. and cleans them up. Requests that mutate global
  server state (keyboard/pointer control, hosts, screensaver, ...) are
  deliberately not exercised — see [xts-coverage.md](xts-coverage.md).
- Today this runs under CI alongside Xorg's XTS; the intent is to grow coverage
  (see TODO) until it can stand in for it.
