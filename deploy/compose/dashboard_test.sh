#!/bin/sh
# Checks the dashboard against the metrics siltide actually exports, so a
# renamed metric cannot leave a panel quietly drawing nothing.
#
#   sh deploy/compose/dashboard_test.sh
#
# It starts siltide on the simulated fleet, reads the names out of /metrics,
# and looks for every siltide_ name the dashboard queries.
set -eu

here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
bin="${SILTIDE:-$root/bin/siltide}"
[ -x "$bin" ] || { echo "no binary at $bin: run make build" >&2; exit 1; }

port=19821
state="$(mktemp -d)"
trap 'kill "$pid" 2>/dev/null || true; rm -rf "$state"' EXIT
SILTIDE_STATE_DIR="$state" "$bin" --demo --service --listen "127.0.0.1:$port" >/dev/null 2>&1 &
pid=$!

i=0
while [ "$i" -lt 50 ]; do
	curl -fsS -o /dev/null "http://127.0.0.1:$port/healthz" 2>/dev/null && break
	i=$((i + 1))
	sleep 0.2
done

exported="$(curl -fsS "http://127.0.0.1:$port/metrics" | sed -n 's/^# HELP \([a-z_]*\) .*/\1/p' | sort -u)"
wanted="$(grep -o 'siltide_[a-z_]*' "$here/grafana/dashboards/siltide.json" | sort -u)"

missing=""
for m in $wanted; do
	echo "$exported" | grep -qx "$m" || missing="$missing $m"
done

if [ -n "$missing" ]; then
	echo "the dashboard queries metrics siltide does not export:$missing" >&2
	exit 1
fi
echo "dashboard: every one of $(echo "$wanted" | wc -l | tr -d ' ') metrics is exported"
