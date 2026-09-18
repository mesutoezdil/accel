#!/bin/sh
# The social preview card: assets/og-home.png, 1200x630, which is what a link
# to the site shows in a chat or a search result.
#
#   scripts/shots/og.sh
#
# It is built from a real capture rather than drawn, for the same reason the
# hero is: the picture should be the program's own output.
set -eu
chrome="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
[ -x "$chrome" ] || { echo "og.sh: no Chrome at $chrome" >&2; exit 1; }

root="$(cd "$(dirname "$0")/../.." && pwd)"
shot="${1:-$root/assets/overview.png}"
[ -f "$shot" ] || { echo "og.sh: no capture at $shot; run make shots first" >&2; exit 1; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/card.html" <<HTML
<html><head><meta charset="utf-8"><style>
  @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;700&display=swap');
  html,body{margin:0;width:1200px;height:630px;overflow:hidden;background:#0b0a09;
    font-family:'JetBrains Mono',ui-monospace,monospace;color:#ece6db}
  .shot{position:absolute;inset:170px 0 -80px 60px;background:url(file://$shot) no-repeat top left;
    background-size:1320px auto;border-radius:12px 0 0 0;box-shadow:0 -10px 60px rgba(0,0,0,.6)}
  .fade{position:absolute;inset:0;background:linear-gradient(180deg,#0b0a09 30%,rgba(11,10,9,.72) 55%,rgba(11,10,9,.35) 100%)}
  .text{position:absolute;left:60px;top:54px;z-index:2}
  h1{margin:0;font-size:58px;letter-spacing:-1px}
  h1 span{color:#ff7a1a}
  p{margin:14px 0 0;font-size:25px;color:#9c9186;max-width:1000px;line-height:1.35}
</style></head><body>
  <div class="shot"></div><div class="fade"></div>
  <div class="text">
    <h1><span>siltide</span> finds every accelerator in the machine.</h1>
    <p>GPUs, NPUs and thirteen other kinds, in one terminal: who holds each device,
       how hot it runs, and what it did ten minutes ago.</p>
  </div>
</body></html>
HTML

"$chrome" --headless=new --disable-gpu --hide-scrollbars --window-size=1200,630 \
	--screenshot="$root/assets/og-home.png" "file://$tmp/card.html" >/dev/null 2>&1
echo "assets/og-home.png"
