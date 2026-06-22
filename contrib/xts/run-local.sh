#!/usr/bin/env bash
# Run XTS against the X server ALREADY running on $DISPLAY (no server spawned).
#
# WARNING: the live tests are DESTRUCTIVE on a real session - they grab the
# server, warp the pointer, and change pointer/screensaver/access-control
# settings. Do NOT run this against a desktop you care about; prefer run-xvfb.sh
# (headless) or run-xephyr.sh / run-xnest.sh (nested, isolated).
#
# Pass --yes (or set XTS_YES=1) to skip the confirmation prompt.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${DISPLAY:?no DISPLAY set}"

echo "WARNING: XTS is destructive and will run against DISPLAY=$DISPLAY" >&2
if [ "${XTS_YES:-}" != 1 ] && [[ " $* " != *" --yes "* ]]; then
	read -rp "Continue? [y/N] " ans || ans=n
	case "$ans" in
	y | Y | yes | YES) ;;
	*) echo "aborted." >&2; exit 1 ;;
	esac
fi
exec "$here/run-xts.sh" none
