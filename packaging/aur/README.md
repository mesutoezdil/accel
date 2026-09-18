# The AUR package

`siltide-bin` installs the published binary, with the completions and the man
page generated from that same binary so they cannot describe another version.

The package is pushed by `.github/workflows/aur-publish.yml`, which the release
workflow calls after a stable release is public, and which can also be run by
hand for any version that is already published. Running it by hand is the point
of it being a separate workflow: a publish that failed should not need another
release to retry.

## What it needs once

An SSH key registered with an AUR account that has write access to
`siltide-bin`, stored as the `AUR_SSH_KEY` secret. The push is done with plain
git rather than a third-party action on purpose: a PKGBUILD runs code on a
user's machine at build time, so write access to it is not something to hand
to someone else's supply chain for twenty lines of shell.

## The AUR holds one version

`pkgver` is a scalar and users install whatever it currently points at, so
there is no back catalogue to fill in. The workflow takes a single version.
