# Installing siltide

Every release and every push to `main` (pre-release `vX.Y.Z-main.N`) publishes binaries, packages, and images.

```sh
# Download the installer, then run it. Two steps rather than curl | sh, so
# you can read the script first. It picks the build for this machine, checks
# it against the published checksums, and installs into /usr/local/bin
# (or ~/.local/bin if that needs a password).
curl -fsSLO https://raw.githubusercontent.com/mesutoezdil/siltide/main/packaging/install/install.sh
sh install.sh

# By hand, if you would rather see every step. The first two lines work out
# the build for THIS machine: siltide-linux-amd64 on an Apple laptop gives
# "exec format error", which is the machine telling you it is not its binary.
os=$(uname -s | tr '[:upper:]' '[:lower:]')                 # linux or darwin
arch=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')  # amd64 or arm64
tag=$(curl -fsSL https://api.github.com/repos/mesutoezdil/siltide/releases | grep -m1 '"tag_name"' | cut -d '"' -f4)
base="https://github.com/mesutoezdil/siltide/releases/download/$tag"
curl -fsSLO "$base/siltide-$os-$arch"
curl -fsSLO "$base/checksums.txt"
shasum -a 256 -c checksums.txt --ignore-missing   # sha256sum -c on Linux
chmod +x "siltide-$os-$arch"
sudo install "siltide-$os-$arch" /usr/local/bin/siltide

# A binary downloaded through a browser rather than curl is quarantined by
# macOS until the builds are signed:
#   xattr -d com.apple.quarantine /usr/local/bin/siltide

# deb or rpm, with completions and the man page
sudo dpkg -i siltide_*_amd64.deb   # or: sudo rpm -i siltide-*.x86_64.rpm

# Homebrew (macOS and Linux)
brew install mesutoezdil/tap/siltide

# Go
go install github.com/mesutoezdil/siltide@latest

# Nix, without installing anything
nix run github:mesutoezdil/siltide -- --demo

# Container: headless collector with the API and /metrics on port 9800.
# --pid=host lets it see host processes, not just its own container.
docker run --rm -p 9800:9800 --gpus all --pid=host \
  -e NVIDIA_DRIVER_CAPABILITIES=utility ghcr.io/mesutoezdil/siltide:latest
```

A [systemd unit](../deploy/systemd/siltide.service), a [Kubernetes DaemonSet](../deploy/kubernetes/daemonset.yaml) and a
[compose stack with Prometheus and Grafana](../deploy/compose/) are in `deploy/`.
What siltide costs to run, and the scripts that measure it, are in [docs/PERFORMANCE.md](PERFORMANCE.md).

## Verifying a download

Every release publishes `checksums.txt` beside the binaries. The install
script checks the download against it before writing anything; by hand it is:

```sh
sha256sum -c checksums.txt --ignore-missing   # shasum -a 256 on macOS
```

Each artifact also carries a build-provenance attestation, which says which
workflow at which commit produced it:

```sh
gh attestation verify "siltide-$os-$arch" --repo mesutoezdil/siltide
```

## Keeping it up to date

```sh
siltide --update
```

It resolves the newest release, downloads the build for this machine, and
checks its SHA-256 against the `checksums.txt` published with the release
*before* anything is written. The last step is a rename, not a copy, so an
interrupted update cannot leave half a binary on your PATH. A build whose
version carries a pre-release suffix stays on pre-releases; everything else
follows stable.

If the binary came from a package manager, `--update` refuses and says which
one to use instead: replacing a packaged file leaves the package database
describing a version that is no longer there, and the next upgrade reverts it.

- **deb, rpm, apk, Arch, Homebrew, Nix**: the package manager does it.
- **The install script**: `siltide --update`, or run the script again.
- **`go install`**: run it again with `@latest`.
- **The container**: pull the tag again.

## Uninstalling

```sh
sudo rm /usr/local/bin/siltide          # or ~/.local/bin/siltide
sudo apt remove siltide                  # or dnf/apk/pacman, if a package was used
brew uninstall siltide                   # Homebrew

rm -rf ~/.config/siltide                 # config, themes
rm -rf ~/.local/state/siltide            # history, bookmarks, the session, the log
```

Nothing else is left behind: siltide installs no service, writes nothing
outside those two directories, and starts no daemon.
