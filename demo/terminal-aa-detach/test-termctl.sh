#!/usr/bin/env bash
#
# test-termctl.sh — manual smoke test for the termctl-backed
# demo/terminal-aa-detach in a real X session (DISPLAY=:0.0).
#
# Exercises:
#   * detached start (no window) + signal-driven attach/detach via SIGUSR1/2
#   * window-close auto-detach + re-attach
#   * the caller-side registry + "ctl" subcommand (stand-in for starfleetctl):
#     a second process drives the running terminal by NAME over OpenPipe.
#
# Each step waits for your <enter> so you can observe the state.
#
set -u

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DEMO_DIR="$REPO_ROOT/demo/terminal-aa-detach"
BIN="$DEMO_DIR/terminal-aa-detach"

echo "== building demo =="
( cd "$REPO_ROOT" && go build -o "$BIN" ./demo/terminal-aa-detach/ ) || {
	echo "build failed"; exit 1
}
echo "built: $BIN"

pause() {
	local msg="${1:-press <enter> to continue}"
	read -r -p "$msg"
}

echo
echo "=================================================="
echo "TEST 1: signal-driven attach/detach (SIGUSR1/SIGUSR2)"
echo "=================================================="
"$BIN" run --detached --name test1 &
PID=$!
echo "started pid=$PID (no window yet)"
pause "-> send SIGUSR2 to ATTACH (window should appear)"
kill -USR2 "$PID"
pause "-> send SIGUSR1 to DETACH (window should vanish, shell alive)"
kill -USR1 "$PID"
pause "-> send SIGUSR2 again to RE-ATTACH"
kill -USR2 "$PID"
pause "-> now CLOSE the window (should auto-detach, shell survives)"
pause "-> send SIGUSR2 once more to RE-ATTACH after window-close"
kill -USR2 "$PID"
pause "-> stopping test1 (SIGTERM)"
kill -TERM "$PID"
wait "$PID" 2>/dev/null
echo "test1 done"

echo
echo "=================================================="
echo "TEST 2: caller-side registry + 'ctl' by name (OpenPipe)"
echo "=================================================="
# This simulates starfleetctl: a terminal is started and recorded in the demo
# registry; a SEPARATE process (the 'ctl' subcommand) drives it by name.
"$BIN" run --detached --name test2 &
PID=$!
echo "started pid=$PID, name=test2"
pause "-> ctl test2 attach (window should appear)"
"$BIN" ctl test2 attach
pause "-> ctl test2 detach (window should vanish, shell alive)"
"$BIN" ctl test2 detach
pause "-> ctl test2 attach again"
"$BIN" ctl test2 attach
pause "-> ctl test2 stop (shell killed, registry entry removed)"
"$BIN" ctl test2 stop
wait "$PID" 2>/dev/null
echo "test2 done; registry entry should be gone:"
grep -q test2 "$REPO_ROOT"/../*/termctl-demo-registry.txt 2>/dev/null && echo "STILL PRESENT" || echo "removed (or no registry file)"

echo
echo "ALL TESTS COMPLETE"
