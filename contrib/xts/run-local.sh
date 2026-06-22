#!/usr/bin/env bash
# Run XTS against the X server ALREADY running on $DISPLAY (no server spawned).
#
# WARNING: the live tests are DESTRUCTIVE on a real session - they grab the
# server, warp the pointer, and change pointer/screensaver/access-control
# settings. Do NOT run this against a desktop you care about; prefer run-xvfb.sh
# (headless) or run-xephyr.sh / run-xnest.sh (nested, isolated).
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${DISPLAY:?no DISPLAY set}"

echo "running destructive XTS against DISPLAY=$DISPLAY" >&2
exec "$here/run-xts.sh" none
