#!/bin/bash
# Self-contained smoke test for the XEmbed tab manager (demo/tabbed).
# Builds the binaries, starts Xvfb via simple-xinit, runs tabbed with two
# terminals, verifies XEmbed handshake and tab switching via Alt+Tab/Alt+Q.
#
# Usage: ./demo/tabbed/smoke.sh
#
# Environment:
#   SIMPLE_XINIT  path to simple-xinit binary (default: search workspace)
#   XVFB          path to Xvfb binary (default: /bin/Xvfb)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# The workspace root is typically the parent of the repo's _WORK_ dir.
WS_ROOT="${WS_ROOT:-$ROOT/../../../..}"
BUILD_DIR="$ROOT/_WORK_/smoke-bin"
OUT="$BUILD_DIR/smoke.log"
SIMPLE_XINIT="${SIMPLE_XINIT:-}"
XVFB="${XVFB:-/bin/Xvfb}"
DISP=":99"

log() { echo "$@" | tee -a "$OUT"; }
die() { log "FAIL: $*"; exit 1; }

mkdir -p "$BUILD_DIR"
: > "$OUT"

# Locate simple-xinit if not provided via env.
if [ -z "$SIMPLE_XINIT" ]; then
    for p in \
        "$WS_ROOT/_WORK_/xserver-master/build/xlibre/xserver/test/simple-xinit" \
        "$WS_ROOT/_WORK_/xserver-container/build/xlibre/xserver/test/simple-xinit" \
        "$ROOT/_WORK_/xserver-master/build/xlibre/xserver/test/simple-xinit" \
        "$ROOT/_WORK_/xserver-container/build/xlibre/xserver/test/simple-xinit" \
        "/usr/local/bin/simple-xinit" \
        "/usr/bin/simple-xinit"; do
        if [ -x "$p" ]; then
            SIMPLE_XINIT="$p"
            break
        fi
    done
fi
[ -x "$SIMPLE_XINIT" ] || die "simple-xinit not found (set SIMPLE_XINIT)"

# Build the two binaries.
log "=== building demo/tabbed and demo/terminal ==="
cd "$ROOT"
go build -o "$BUILD_DIR/tabbed" ./demo/tabbed || die "build tabbed failed"
go build -o "$BUILD_DIR/terminal" ./demo/terminal || die "build terminal failed"
log "built: $BUILD_DIR/tabbed  $BUILD_DIR/terminal"

# Write the client script to a file (executed by simple-xinit via shell).
CLIENT="$BUILD_DIR/client.sh"
cat > "$CLIENT" <<'CLIENT_EOF'
#!/bin/bash
set -euo pipefail
export DISPLAY="$DISP"
BIN="$BUILD_DIR"
OUT="$OUT"
DISP="$DISP"

log() { echo "$@" | tee -a "$OUT"; }
die() { log "FAIL: $*"; exit 1; }

log "=== client starting on $DISP ==="

# Launch tabbed with two independent terminal processes.
"$BIN/tabbed" "$BIN/terminal" "$BIN/terminal" &
TBPID=$!
sleep 3

log "=== root window tree ==="
xwininfo -display "$DISP" -root -tree >> "$OUT" 2>&1

log "=== _XEMBED_INFO on windows ==="
for w in $(xwininfo -display "$DISP" -root -tree 2>/dev/null | \
           grep -oE '0x[0-9a-f]{5,}' | sort -u); do
    info=$(xprop -display "$DISP" -id "$w" _XEMBED_INFO 2>/dev/null)
    [ -n "$info" ] && echo "$w : $info" >> "$OUT" 2>&1
done

n_info=$(grep -cE "_XEMBED_INFO(\(CARDINAL\))? = " "$OUT" || true)
log "=== _XEMBED_INFO count: $n_info (expect >= 2) ==="
[ "$n_info" -ge 2 ] || die "expected >= 2 _XEMBED_INFO properties, got $n_info"

# Mapped (= visible) windows before switching.
mapped() {
    for w in $(xwininfo -display "$DISP" -root -tree 2>/dev/null | \
               grep -oE '0x[0-9a-f]{5,}' | sort -u); do
        xwininfo -display "$DISP" -id "$w" 2>/dev/null | \
            grep -qi "Map State: IsViewable" && echo "$w"
    done
}

log "=== mapped window ids before Alt+Tab ==="
mapped >> "$OUT" 2>&1

# Tab switching: synthetic Alt+Tab to switch tabs.
CONT=$(xdotool search --name tabbed 2>/dev/null | head -1)
if [ -n "$CONT" ]; then
    xdotool key --window "$CONT" alt+Tab
    sleep 1
else
    log "WARN: tabbed container not found for tab switching"
fi
log "=== mapped window ids after single Alt+Tab (expect other terminal visible) ==="
mapped >> "$OUT" 2>&1

# Quit with Alt+Q.
if [ -n "$CONT" ]; then
    xdotool key --window "$CONT" alt+Q
    sleep 1
fi
log "=== after Alt+Q: root tree ==="
xwininfo -display "$DISP" -root -tree >> "$OUT" 2>&1

kill "$TBPID" 2>/dev/null || true
exit 0
CLIENT_EOF

chmod +x "$CLIENT"

# Export variables needed by client script.
export BUILD_DIR OUT DISP

log "=== launching Xvfb + test client via simple-xinit ==="
"$SIMPLE_XINIT" "$CLIENT" -- "$XVFB" "$DISP" -screen 0 1024x768x24 2>&1 | tee -a "$OUT"
RC=$?
if [ "$RC" -ne 0 ]; then
    die "simple-xinit exited with rc=$RC"
fi

# Final verification
n_info=$(grep -cE "_XEMBED_INFO(\(CARDINAL\))? = " "$OUT" || true)
if [ "$n_info" -lt 2 ]; then
    die "final _XEMBED_INFO count=$n_info < 2"
fi

# Check that tab switching occurred: visible terminal changed after Alt+Tab.
before=$(sed -n '/mapped window ids before Alt/,/mapped window ids after/p' "$OUT" | grep -E '^0x[0-9a-f]{5,}$' | sort)
after=$(sed -n '/mapped window ids after single Alt/,/after Alt+Q/p' "$OUT" | grep -E '^0x[0-9a-f]{5,}$' | sort)
if [ "$before" = "$after" ]; then
    die "tab switching did not change visible terminal windows (before=$before after=$after)"
fi
log "=== tab switching verified: visible terminal changed ==="

log "=== ALL CHECKS PASSED ==="
exit 0