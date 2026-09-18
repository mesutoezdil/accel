#!/bin/sh
# Builds the animated demo README.md and the site hero embed: renders the
# storyboard to SVG frames, rasterizes each with headless Chrome so the type
# looks the same everywhere, and assembles them into a GIF.
#
#	scripts/shots/film.sh [frame dir] [output gif]
set -eu
chrome="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
[ -x "$chrome" ] || { echo "film.sh: no Chrome at $chrome" >&2; exit 1; }
dir="${1:-build/film}"
out="${2:-assets/demo.gif}"
fps="${FPS:-10}"

go run ./scripts/shots -film "$dir" -width "${COLS:-160}" -height "${ROWS:-42}"

rm -f "$dir"/frame-*.png
first=$(ls "$dir"/frame-*.svg | head -1)
w=$(sed -n '1s/.* width="\([0-9]*\)".*/\1/p' "$first")
h=$(sed -n '1s/.* height="\([0-9]*\)".*/\1/p' "$first")
echo "rasterizing $(ls "$dir"/frame-*.svg | wc -l | tr -d ' ') frames at ${w}x${h}"
case "$dir" in /*) abs="$dir" ;; *) abs="$PWD/$dir" ;; esac
ls "$abs"/frame-*.svg | xargs -P 4 -I@ sh -c \
	'"$0" --headless=new --disable-gpu --hide-scrollbars --window-size="$1,$2" --screenshot="${3%.svg}.png" "file://$3" >/dev/null 2>&1' \
	"$chrome" "$w" "$h" @

go run ./scripts/gif -out "$out" -fps "$fps" "$abs"/frame-*.png
