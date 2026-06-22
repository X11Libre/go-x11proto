#!/usr/bin/env bash
# Run XTS against a freshly spawned Xvfb (headless). Safe: touches no session.
# Extra args override the default screen, e.g. run-xvfb.sh -screen 0 800x600x16
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ $# -gt 0 ]; then
	exec "$here/run-xts.sh" Xvfb "$@"
fi
exec "$here/run-xts.sh" Xvfb -screen 0 1280x1024x24
