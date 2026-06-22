#!/usr/bin/env bash
#
# run-xts.sh - run the go-x11proto XTS (live X11 wire-protocol round-trip tests)
# against a given X server binary.
#
# Intended for the XLibre xserver CI: build the server, then point this at it to
# exercise the core protocol end to end (the eventual replacement for Xorg XTS).
#
# Usage:
#   run-xts.sh [SERVER_BINARY] [SERVER_ARGS...]
#
#   SERVER_BINARY  path to (or $PATH name of) an X server supporting -displayfd
#                  (Xvfb, Xephyr, Xorg, Xwayland - anything from the xserver
#                  tree). Default "Xvfb". May also be set via $XTS_XSERVER.
#   SERVER_ARGS    extra server arguments besides -displayfd (which the harness
#                  adds itself). Default "-screen 0 1280x1024x24". May also be
#                  set via $XTS_XSERVER_ARGS.
#
# Environment:
#   GO            go binary to use (default: go)
#   GOTESTFLAGS   flags passed to `go test` (default: -count=1 -v)
#
# The harness (xts/main_test.go) starts the server on a private display via
# -displayfd, so this never touches an existing X session.
set -euo pipefail

# repo root = two levels up from this script (contrib/xts/run-xts.sh)
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

server="${1:-${XTS_XSERVER:-Xvfb}}"
[ $# -gt 0 ] && shift || true
if [ $# -gt 0 ]; then
	args="$*"
else
	args="${XTS_XSERVER_ARGS:--screen 0 1280x1024x24}"
fi

# resolve a server given as a relative path before we change directory
if [[ "$server" == */* ]]; then
	server="$(cd "$(dirname "$server")" && pwd)/$(basename "$server")"
fi

echo "xts: server = ${server}"
echo "xts: args   = -displayfd 1 ${args}"
echo "xts: module = ${root}"

export XTS_XSERVER="${server}"
export XTS_XSERVER_ARGS="${args}"

cd "${root}"
exec ${GO:-go} test ${GOTESTFLAGS:--count=1 -v} ./xts/...
