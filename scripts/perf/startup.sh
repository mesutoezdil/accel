#!/bin/sh
# How long siltide takes to put a first view on the screen.
#
#   scripts/perf/startup.sh [runs]
#
# Two numbers, because they answer different questions. The simulated fleet
# measures siltide itself: process start, one collection, one render, with no
# hardware in the way. The real run adds vendor detection, which on a machine
# with fifteen vendors to probe is most of it, and which is the number someone
# actually waits for.
#
# The median is reported rather than the mean: one run that lands on a busy
# moment should not move the figure.
set -eu

runs="${1:-20}"
bin="${SILTIDE:-./bin/siltide}"
[ -x "$bin" ] || { echo "no binary at $bin: run make build, or set SILTIDE" >&2; exit 1; }

median() { sort -n | awk '{ a[NR] = $1 } END { printf "%.3f\n", (NR % 2) ? a[(NR + 1) / 2] : (a[NR / 2] + a[NR / 2 + 1]) / 2 }'; }

time_one() { # time_one <args...>, wall seconds
	# /usr/bin/time -p prints "real 0.03" on both Linux and macOS, which is
	# the one timing format that needs no shell builtin and no second process
	# whose own startup would land in the measurement.
	LC_ALL=C /usr/bin/time -p "$@" 2>&1 >/dev/null | awk '/^real/ { print $2 }'
}

echo "siltide: $("$bin" --version)"
echo "runs:    $runs"

sim="$(i=0; while [ "$i" -lt "$runs" ]; do time_one "$bin" --demo --once; i=$((i + 1)); done | median)"
real="$(i=0; while [ "$i" -lt "$runs" ]; do time_one "$bin" --once; i=$((i + 1)); done | median)"
detect="$("$bin" --diagnose | sed -n 's/^ *detection *//p')"

echo
echo "first rendered snapshot, simulated fleet: ${sim}s (median of $runs)"
echo "first rendered snapshot, this machine:    ${real}s (median of $runs)"
echo "of which vendor detection:                $detect"
