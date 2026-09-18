#!/bin/sh
# Holds every page of the site to the same shape: the shared stylesheet and
# script, a canonical link, a social preview, and the same navigation. A site
# that grows one page at a time drifts otherwise, and nobody notices until a
# link in a chat shows nothing.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
fail=0
say() { echo "site: $*" >&2; fail=1; }

for page in "$root"/site/*.html; do
	name="$(basename "$page")"
	for want in 'rel="stylesheet" href="styles.css"' 'src="site.js"' 'rel="canonical"' \
		'property="og:title"' 'property="og:image"' 'name="twitter:card"' '<title>'; do
		grep -q "$want" "$page" || say "$name has no $want"
	done
	# Every page carries the same destinations, so the navigation cannot rot
	# on the pages nobody edited.
	for link in 'tui.html' 'guides.html' 'compare.html' 'faq.html' 'download.html'; do
		grep -q "href=\"$link\"" "$page" || say "$name does not link to $link"
	done
	# and every page offers a way to install, wherever it sits
	grep -qE 'href="(index\.html)?#install"|href="download\.html"' "$page" ||
		say "$name has no route to installing siltide"
	# Relative links have to point at something that exists.
	for target in $(sed -n 's/.*href="\([a-z0-9._-]*\.html\)".*/\1/p' "$page" | sort -u); do
		[ -f "$root/site/$target" ] || say "$name links to $target, which is not there"
	done
done

[ -f "$root/site/robots.txt" ] || say "no robots.txt"
[ -f "$root/assets/og-home.png" ] || say "no social preview image"

[ "$fail" = 0 ] && echo "site: $(ls "$root"/site/*.html | wc -l | tr -d ' ') pages, all consistent"
exit "$fail"
