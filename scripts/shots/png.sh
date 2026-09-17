#!/bin/sh
# Rasterizes the screenshots README.md embeds (the names given, or the
# fleet set), at 2x, with headless Chrome, so they look the same in every
# viewer. The rest stay SVG.
set -eu
chrome="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
[ -x "$chrome" ] || { echo "png.sh: no Chrome at $chrome, skipping" >&2; exit 0; }
[ $# -gt 0 ] || set -- overview devices history links health
for name in "$@"; do
	f="assets/$name.svg"
	w=$(sed -n '1s/.* width="\([0-9]*\)".*/\1/p' "$f")
	h=$(sed -n '1s/.* height="\([0-9]*\)".*/\1/p' "$f")
	"$chrome" --headless=new --disable-gpu --hide-scrollbars --force-device-scale-factor=2 \
		--window-size="$w,$h" --screenshot="assets/$name.png" "file://$PWD/$f" >/dev/null 2>&1
	echo "assets/$name.png"
done
