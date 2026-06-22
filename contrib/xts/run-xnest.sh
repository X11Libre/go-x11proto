#!/usr/bin/env bash
# Run XTS against a spawned Xnest - a nested server shown as a window on your
# current $DISPLAY. The tests run against the nested server, so your own session
# is untouched. Extra args override the default geometry.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${DISPLAY:?Xnest needs a parent DISPLAY to open its window}"
if [ $# -gt 0 ]; then
	exec "$here/run-xts.sh" Xnest "$@"
fi
exec "$here/run-xts.sh" Xnest -geometry 1280x1024
