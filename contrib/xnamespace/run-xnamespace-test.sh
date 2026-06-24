#!/usr/bin/env bash
#
# run-xnamespace-test.sh - run the go-x11proto X-NAMESPACE conformance test
# against a given X server binary, in the spirit of run-xts.sh.
#
# Intended for the XLibre xserver CI: build a server with the namespace
# extension (CONFIG_NAMESPACE), then point this at it to exercise the extension
# end to end. It spawns the server on a private display with a generated config
# that grants the (unauthenticated) test client `superpower`, so the privileged
# requests are reachable, runs the TAP test tool in both client byte orders, and
# cleans everything up.
#
# Usage:
#   run-xnamespace-test.sh [SERVER_BINARY] [SERVER_ARGS...]
#
#   SERVER_BINARY  path to (or $PATH name of) an X server built with the
#                  namespace extension and supporting -displayfd and -namespace
#                  (e.g. the xlibre Xvfb). Default "Xvfb"; also $XNS_XSERVER.
#   SERVER_ARGS    extra server arguments besides -displayfd / -namespace (which
#                  the harness adds). Default "-screen 0 1280x1024x24"; also
#                  $XNS_XSERVER_ARGS.
#
# Environment:
#   GO              go binary (default: go)
#   XNS_TEST_FLAGS  extra flags for xnamespace-test (e.g. -v)
#
# Exit status is the test tool's: 0 if every test passes in both byte orders.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

server="${1:-${XNS_XSERVER:-Xvfb}}"
[ $# -gt 0 ] && shift || true
if [ $# -gt 0 ]; then
	args="$*"
else
	args="${XNS_XSERVER_ARGS:--screen 0 1280x1024x24}"
fi

# resolve a server given as a relative path before anything else
if [[ "$server" == */* ]]; then
	server="$(cd "$(dirname "$server")" && pwd)/$(basename "$server")"
fi

tmp="$(mktemp -d)"
srv=""
cleanup() {
	[ -n "$srv" ] && kill "$srv" 2>/dev/null || true
	[ -n "$srv" ] && wait "$srv" 2>/dev/null || true
	rm -rf "$tmp"
}
trap cleanup EXIT

# The big-endian pass needs the server to accept byte-swapped clients, which
# modern xservers refuse by default; +byteswappedclients re-enables it.
echo "xnamespace-test: server = ${server}"
echo "xnamespace-test: args   = -displayfd 3 -namespace <config> +byteswappedclients ${args}"

# build the test tool
( cd "$root" && ${GO:-go} build -o "$tmp/xnamespace-test" ./cmd/xnamespace-test/ )

# config: grant the anonymous namespace superpower, so the test client (which
# connects without an auth cookie and therefore lands in `anon`) may use the
# privileged extension. Also define a child namespace to make the listing
# non-trivial.
cat > "$tmp/xnamespace.conf" <<'EOF'
namespace anon
  superpower

namespace demo root
  allow shape
EOF

# launch the server on a server-assigned display; it writes the display number
# to fd 3, which we capture in a file.
dispfile="$tmp/display"
: > "$dispfile"
# shellcheck disable=SC2086
"$server" -displayfd 3 -namespace "$tmp/xnamespace.conf" +byteswappedclients $args \
	3>"$dispfile" >"$tmp/server.log" 2>&1 &
srv=$!

# wait until the server publishes its display number (or dies)
dnum=""
for _ in $(seq 1 100); do
	if [ -s "$dispfile" ]; then
		dnum="$(tr -d '[:space:]' < "$dispfile")"
		break
	fi
	if ! kill -0 "$srv" 2>/dev/null; then
		echo "xnamespace-test: server exited before becoming ready:" >&2
		cat "$tmp/server.log" >&2
		exit 1
	fi
	sleep 0.1
done
if [ -z "$dnum" ]; then
	echo "xnamespace-test: timed out waiting for the server display" >&2
	cat "$tmp/server.log" >&2
	exit 1
fi

export DISPLAY=":$dnum"
echo "xnamespace-test: DISPLAY=${DISPLAY}"

rc=0
echo "# ===== client byte order: little-endian ====="
"$tmp/xnamespace-test" ${XNS_TEST_FLAGS:-} || rc=1
echo "# ===== client byte order: big-endian ====="
"$tmp/xnamespace-test" -be ${XNS_TEST_FLAGS:-} || rc=1

exit "$rc"
