#!/bin/sh
# Resident memory of siltide while it runs.
#
#   scripts/perf/memory.sh [seconds] [--service]
#
# Samples RSS every two seconds and reports the median, because the first
# reading is taken before the history buffers have filled and flatters the
# number. siltide is a single process with no helpers, so one process is the
# whole footprint: nothing here has to chase a process tree.
set -eu

secs="${1:-30}"
shift 2>/dev/null || true
bin="${SILTIDE:-./bin/siltide}"
[ -x "$bin" ] || { echo "no binary at $bin: run make build, or set SILTIDE" >&2; exit 1; }

state="$(mktemp -d)"
trap 'kill "$pid" 2>/dev/null || true; rm -rf "$state"' EXIT

if [ "${1:-}" = "--service" ]; then
	SILTIDE_STATE_DIR="$state" "$bin" --service --listen 127.0.0.1:19812 >/dev/null 2>&1 &
	what="headless collector"
else
	# The interface needs a terminal, so it gets one and is told to draw into it.
	SILTIDE_STATE_DIR="$state" script -q /dev/null "$bin" --demo >/dev/null 2>&1 &
	what="interface"
fi
pid=$!
sleep 3

i=0
samples=""
while [ "$i" -lt "$secs" ]; do
	# The interface is started under `script` so it has a terminal to draw
	# into, which makes it a child rather than the process we hold; sum the
	# pid and anything descended from it so either shape is measured whole.
	rss="$(ps -eo pid=,ppid=,rss= 2>/dev/null |
		awk -v p="$pid" '$1 == p || $2 == p { sum += $3 } END { print sum }')"
	[ -n "$rss" ] && [ "$rss" != "0" ] || break
	samples="$samples
$(awk -v k="$rss" 'BEGIN { printf "%.1f", k / 1024 }')"
	sleep 2
	i=$((i + 2))
done

echo "$what, resident memory over ${secs}s:"
echo "$samples" | sed '/^$/d' | sort -n |
	awk '{ a[NR] = $1 } END { printf "  median %.1f MB   min %.1f MB   max %.1f MB   (%d samples)\n", (NR % 2) ? a[(NR + 1) / 2] : (a[NR / 2] + a[NR / 2 + 1]) / 2, a[1], a[NR], NR }'
