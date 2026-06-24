# Testing the X-NAMESPACE extension in an xserver build (CI integration)

`cmd/xnamespace-test` is a live **conformance test** for the X-NAMESPACE
(Xnamespace) extension, in the spirit of Xorg's XTS or `rendercheck`: it opens a
real connection, drives every extension request and the documented error cases,
and reports [TAP](https://testanything.org/). Because it talks the wire protocol
(via `proto/ext/namespace`) rather than linking a client library, it runs
against any xserver built with the namespace extension (`CONFIG_NAMESPACE`).

The goal is to run it as part of the **XLibre xserver** build/CI, against the
just-built server.

## The privilege catch

The whole extension is reachable only by clients in a `superPower` namespace; to
everyone else `QueryExtension` reports it as absent. A client that connects
without an auth cookie (as this tool does) lands in the builtin `anon`
namespace, which is **not** privileged by default.

The runner therefore generates a tiny config that grants `anon` superpower and
starts the server with it:

```
namespace anon
  superpower

namespace demo root
  allow shape
```

So the test client — unauthenticated, hence in `anon` — can drive the privileged
requests. (In production you would instead match a real auth cookie to a
superpower namespace; granting `anon` is purely a test affordance.)

## One-shot / local

```sh
# against a freshly built Xvfb from an xserver build tree (with CONFIG_NAMESPACE):
contrib/xnamespace/run-xnamespace-test.sh /path/to/xserver/build/hw/vfb/Xvfb

# defaults to the Xvfb on $PATH:
contrib/xnamespace/run-xnamespace-test.sh
```

The runner:

1. builds `cmd/xnamespace-test`,
2. writes the superpower config to a temp dir,
3. launches the server on a private display via `-displayfd` (so it never
   touches an existing session), adding `-namespace <config>` and
   `+byteswappedclients` (the latter so the big-endian pass is accepted),
4. runs the test tool once per **client byte order** (little- and big-endian),
5. kills the server and cleans up, exiting non-zero if any test failed.

| Variable | Default | Meaning |
|----------|---------|---------|
| `XNS_XSERVER` | `Xvfb` | server executable: a `$PATH` name or an absolute/relative path |
| `XNS_XSERVER_ARGS` | `-screen 0 1280x1024x24` | extra args (besides `-displayfd`, `-namespace`, `+byteswappedclients`) |
| `GO` | `go` | Go toolchain to build the tool |
| `XNS_TEST_FLAGS` | _(empty)_ | extra flags for `xnamespace-test`, e.g. `-v` |

`xnamespace-test` can also be run directly against an already-running,
namespace-enabled `$DISPLAY`:

```sh
go run ./cmd/xnamespace-test -v          # little-endian client
go run ./cmd/xnamespace-test -be         # big-endian client
```

## What it checks

20 tests per byte order: version negotiation; `GetClientNamespace`; the builtin
`root`/`anon` namespaces and `root` immutability; create / query / set-flags /
delete; auth-token add / list / remove; the transient attribute; and the
documented error cases — `BadName` (duplicate / empty / illegal name / missing
namespace), `BadValue` (reserved capability / attribute bit), and `BadAccess`
(mutating a builtin namespace).

## GitHub Actions (in the xserver repo)

Add a step to the job that already builds the server:

```yaml
      # ... after the xserver meson build, binaries in ./build ...
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - name: Check out go-x11proto
        uses: actions/checkout@v4
        with:
          repository: X11Libre/go-x11proto
          path: go-x11proto
      - name: Run X-NAMESPACE conformance against the built Xvfb
        run: ./go-x11proto/contrib/xnamespace/run-xnamespace-test.sh "$GITHUB_WORKSPACE/build/hw/vfb/Xvfb"
```

## Meson test integration (in the xserver repo)

TAP output makes it a natural meson test:

```meson
go = find_program('go', required: false)
if go.found() and get_option('xnamespace')
  test('xnamespace',
    files('contrib/xnamespace/run-xnamespace-test.sh'),  # vendored / submoduled go-x11proto
    args: [Xvfb.full_path()],                            # the build's Xvfb target
    protocol: 'tap',
    timeout: 120,
    is_parallel: false,
  )
endif
```

(Adjust the script path and `Xvfb` target reference to match how go-x11proto is
made available in the build — git submodule, subproject, or a CI checkout.)

## Notes

- Needs a Go toolchain (module targets **go 1.22**) and a server built with the
  namespace extension. No X client libraries required.
- The tool cleans up the namespaces it creates (before and after the run), so
  repeated runs are idempotent.
- The extension is **DRAFT v1.0**; this tracks the wire layout in the server's
  `Xext/namespace/namespaceproto.h`.
