#!/usr/bin/env bash
#
# test-termctl.sh — manual smoke test for the termctl-backed
# demo/terminal-aa-detach in a real X session (DISPLAY=:0.0).
#
# What it exercises:
#   * detached start (no window)
#   * SIGUSR2 -> attach (window appears)
#   * SIGUSR1 -> detach (window gone, shell alive)
#   * closing the window -> auto-detach
#   * control pipe mode: --pipe PATH + "attach"/"detach"/"status" commands
#   * TERM_CTRL_FD mode: parent opens a pipe, spawns child with the fd
#
# It does NOT auto-verify pixels; you watch the windows. Each step waits for
# your <enter> so you can observe the state.
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
echo "TEST 1: detached start + signal-driven attach/detach"
echo "=================================================="
"$BIN" --detached --name test1 &
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
echo "TEST 2: control pipe mode (--pipe PATH)"
echo "=================================================="
PIPE=$(mktemp -u /tmp/termctl-test2.XXXXXX)
"$BIN" --detached --name test2 --pipe "$PIPE" &
PID=$!
echo "started pid=$PID, pipe=$PIPE"
pause "-> attach via pipe"
printf 'attach\n' > "$PIPE"
pause "-> status via pipe"
printf 'status\n' > "$PIPE"
pause "-> detach via pipe"
printf 'detach\n' > "$PIPE"
pause "-> attach again via pipe"
printf 'attach\n' > "$PIPE"
pause "-> quit via pipe"
printf 'quit\n' > "$PIPE"
wait "$PID" 2>/dev/null
echo "test2 done; pipe removed by Close: $(ls -l "$PIPE" 2>&1 || true)"

echo
echo "=================================================="
echo "TEST 3: inherited control fd (TERM_CTRL_FD)"
echo "=================================================="
# Parent opens a pipe, passes the write-end fd to the child as TERM_CTRL_FD.
# The child reads commands on that fd. We write commands from the parent.
PIPE3=$(mktemp -u /tmp/termctl-test3.XXXXXX)
mkfifo "$PIPE3"
exec {WFD}>"$PIPE3"
"$BIN" --detached --name test3 --pipe /dev/null &
PID=$!
# Re-exec with fd inheritance: simpler to use a subshell that dups the fd.
# (The demo reads TERM_CTRL_FD as an already-open fd number.)
(
	# open the read end and export its fd number
	exec {RFDrd}<"$PIPE3"
	TERM_CTRL_FD=$RFDrd "$BIN" --detached --name test3 &
	CHILDPID=$!
	echo "started child pid=$CHILDPID reading from fd=$RFDrd"
	sleep 1
	echo "attach" >&$WFD
	sleep 1
	echo "-> child should have attached (window visible)"
	sleep 2
	echo "detach" >&$WFD
	sleep 1
	echo "-> child should have detached"
	sleep 2
	echo "quit" >&$WFD
	wait $CHILDPID 2>/dev/null
)
rm -f "$PIPE3"
echo "test3 done"

echo
echo "ALL TESTS COMPLETE"
