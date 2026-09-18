#!/bin/sh
# Tests for install.sh.
#
#   sh packaging/install/test.sh
#
# An install script is the easiest thing in a repository to leave broken:
# nothing builds it, nothing imports it, and the only person who finds out is
# a stranger following the README. These run it against a fake release served
# from a local directory, with fake `curl` and `uname` ahead of the real ones
# on PATH, so nothing here needs the network.
set -eu

here="$(cd "$(dirname "$0")" && pwd)"
script="$here/install.sh"
[ -f "$script" ] || { echo "install.sh not found next to $0" >&2; exit 1; }

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
pass=0
fail=0

check() { # check <name> <expected substring> <exit code> <command...>
	name="$1" want="$2" code="$3"
	shift 3
	set +e
	out="$("$@" 2>&1)"
	got=$?
	set -e
	if [ "$got" != "$code" ]; then
		echo "FAIL $name: exit $got, want $code"
		echo "$out" | sed 's/^/     /'
		fail=$((fail + 1))
		return
	fi
	case "$out" in
		*"$want"*) pass=$((pass + 1)); echo "ok   $name" ;;
		*) echo "FAIL $name: output does not mention '$want'"; echo "$out" | sed 's/^/     /'; fail=$((fail + 1)) ;;
	esac
}

# A release directory the fake curl serves: one asset, one checksum list.
rel="$work/release"
mkdir -p "$rel"
printf '#!/bin/sh\necho "siltide v9.9.9"\n' > "$rel/siltide-linux-amd64"
chmod +x "$rel/siltide-linux-amd64"
sum="$(sha256sum "$rel/siltide-linux-amd64" 2>/dev/null | cut -d' ' -f1 || shasum -a 256 "$rel/siltide-linux-amd64" | cut -d' ' -f1)"
printf '%s  siltide-linux-amd64\n' "$sum" > "$rel/checksums.txt"
printf '{"tag_name": "v9.9.9"}\n' > "$rel/latest.json"
# The releases list, newest first, as the API returns it.
printf '[{"tag_name": "v9.9.9-main.3", "prerelease": true},{"tag_name": "v9.9.8"}]\n' > "$rel/list.json"

# Fake curl: maps the URLs the script asks for onto that directory.
bin="$work/bin"
mkdir -p "$bin"
cat > "$bin/curl" <<EOF
#!/bin/sh
out=""
url=""
while [ \$# -gt 0 ]; do
  case "\$1" in
    -o) out="\$2"; shift 2 ;;
    -*) shift ;;
    *) url="\$1"; shift ;;
  esac
done
case "\$url" in
  *releases/latest) src="$rel/latest.json" ;;
  *releases\?*) src="$rel/list.json" ;;
  *checksums.txt) src="$rel/checksums.txt" ;;
  *siltide-linux-amd64) src="$rel/siltide-linux-amd64" ;;
  *) exit 22 ;;
esac
[ -f "\$src" ] || exit 22
if [ -n "\$out" ]; then cp "\$src" "\$out"; else cat "\$src"; fi
EOF
cat > "$bin/uname" <<'EOF'
#!/bin/sh
case "$1" in
  -s) echo Linux ;;
  -m) echo x86_64 ;;
  *) echo Linux ;;
esac
EOF
chmod +x "$bin/curl" "$bin/uname"
export PATH="$bin:$PATH"

dest="$work/dest"
check "installs the latest release" "install: $dest/siltide" 0 sh "$script" --dir "$dest"
[ -x "$dest/siltide" ] || { echo "FAIL the binary is not executable"; fail=$((fail + 1)); }
check "a pinned version works too" "9.9.9" 0 sh "$script" --dir "$dest" --version 9.9.9
check "a leading v is accepted" "9.9.9" 0 sh "$script" --dir "$dest" --version v9.9.9
check "--version with no value is refused" "needs a value" 1 sh "$script" --dir "$dest" --version
check "an unknown option is refused" "unknown option" 1 sh "$script" --dir "$dest" --nope
check "--help explains itself" "siltide installer" 0 sh "$script" --help

# A release whose checksum does not match the asset must install nothing.
printf '%s  siltide-linux-amd64\n' "0000000000000000000000000000000000000000000000000000000000000000" > "$rel/checksums.txt"
bad="$work/bad"
check "a bad checksum stops the install" "checksum mismatch" 1 sh "$script" --dir "$bad"
[ -e "$bad/siltide" ] && { echo "FAIL a binary was installed despite the checksum"; fail=$((fail + 1)); }
printf '%s  siltide-linux-amd64\n' "$sum" > "$rel/checksums.txt"

# Before the first stable tag, /releases/latest is a 404 and the only builds
# are pre-releases. The installer has to find them: this was the state of this
# repository, and the script told everyone who ran it to pass --version.
rm -f "$rel/latest.json"
check "it falls back to the newest pre-release" "pre-release 9.9.9-main.3" 0 \
	sh "$script" --dir "$work/pre"
[ -x "$work/pre/siltide" ] || { echo "FAIL nothing was installed from the pre-release"; fail=$((fail + 1)); }
printf '{"tag_name": "v9.9.9"}\n' > "$rel/latest.json"

# With no releases at all it should say so, not ask for a version it cannot
# name.
mv "$rel/list.json" "$rel/list.json.off"
rm -f "$rel/latest.json"
check "no releases at all is reported plainly" "no release found" 1 sh "$script" --dir "$work/none2"
mv "$rel/list.json.off" "$rel/list.json"
printf '{"tag_name": "v9.9.9"}\n' > "$rel/latest.json"

# A machine with no build for it should say so rather than install something else.
cat > "$bin/uname" <<'EOF'
#!/bin/sh
case "$1" in
  -s) echo Plan9 ;;
  -m) echo x86_64 ;;
esac
EOF
chmod +x "$bin/uname"
check "an unsupported system is refused" "no build for Plan9" 1 sh "$script" --dir "$work/none"

echo "$pass passed, $fail failed"
[ "$fail" = 0 ]
