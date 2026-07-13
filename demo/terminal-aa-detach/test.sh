#!/usr/bin/env bash
# Interaktive Testumgebung für terminal-aa-detach auf dem aktuellen X-Server.
set -euo pipefail

BIN="$(dirname "$0")/terminal-aa-detach"

cleanup() {
    echo; echo "=== cleanup ==="
    [ -f /tmp/term-test.pids ] && while read -r pid; do kill "$pid" 2>/dev/null || true; done < /tmp/term-test.pids
    rm -f /tmp/term-p{2,3} /tmp/term-test.pids
    # Close parent shell's write fds if open
    exec 11>&- 2>/dev/null || true
    exec 12>&- 2>/dev/null || true
    echo "done"
}
trap cleanup EXIT INT TERM

echo "=== Build ==="
go build -o "$BIN" .

rm -f /tmp/term-p{2,3}
mkfifo /tmp/term-p2
mkfifo /tmp/term-p3

> /tmp/term-test.pids

# 1: per Signal (no control pipe)
"$BIN" &
echo "$!" >> /tmp/term-test.pids

# 2: --detached, per Pipe (subshell opens READ end on fd 10)
( exec 10</tmp/term-p2; TERM_CTRL_FD=10 "$BIN" --detached ) &
echo "$!" >> /tmp/term-test.pids

# 3: normal, per Pipe (subshell opens READ end on fd 10)
( exec 10</tmp/term-p3; TERM_CTRL_FD=10 "$BIN" ) &
echo "$!" >> /tmp/term-test.pids

# Parent shell opens WRITE ends (unblocks subshells)
exec 11>/tmp/term-p2
exec 12>/tmp/term-p3

sleep 0.5

echo ""
echo "╔══════════════════════════════════════════════╗"
echo "║  3 Terminals laufen                         ║"
echo "╠══════════════════════════════════════════════╣"
echo "║  #1  SIGUSR1=detach  SIGUSR2=attach         ║"
echo "║                                              ║"
echo "║  #2  echo detach >&11                       ║"
echo "║      echo 'attach :0' >&11                  ║"
echo "║      echo status >&11                       ║"
echo "║                                              ║"
echo "║  #3  echo detach >&12                       ║"
echo "║      echo 'attach :0' >&12                  ║"
echo "║      echo status >&12                       ║"
echo "║                                              ║"
echo "║  Alle: echo quit >&11 / >&12                ║"
echo "╚══════════════════════════════════════════════╝"
echo "PIDs: $(cat /tmp/term-test.pids)"
echo ""

wait
