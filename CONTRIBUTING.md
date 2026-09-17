# Contributing

Thanks for helping. Issues and pull requests are welcome; hardware reports most of all, since most vendors are tested only against captured tool output.

## Workflow

Every change starts as an issue, including changes by maintainers, so the reasoning is on record before the code is.

1. Open one issue per problem or feature (bug, hardware report, enhancement, documentation). Say what is wrong or missing and how you would know it is fixed.
2. Branch from `main`, make the change, and open a pull request whose description says which issues it closes (`Closes #12, #13`). Small pull requests that close a few related issues merge fastest.
3. CI must be green on the pull request.
4. A maintainer reviews and replies `LGTM` on the pull request; a change the maintainer wrote gets the same review from a second maintainer when there is one, or a self-review comment that records what was checked.
5. The pull request is merged with a rebase (no merge commits, no squashing away the history) and the branch is deleted.

Nothing is pushed to `main` directly.

## Before a pull request

Every one of these must pass; CI runs the same set on Linux and macOS:

```sh
gofmt -l .                       # prints nothing
go vet ./... && GOOS=linux go vet ./...
go test -race ./...
golangci-lint run ./...          # and once more with GOOS=linux
GOOS=linux CGO_ENABLED=0 go build .
```

`make check` runs the lot.

## What a change needs

- A metric is either measured or absent. Never write 0 for "unknown".
- User-facing text says "accelerator" or "device", or names the family (NPU, XPU, MLU), when it means every vendor. "GPU" is for GPUs.
- Vendor tool output used as a test fixture must be real. Add it under `internal/provider/smi/testdata/` and record where it came from, with the license of that source, in `testdata/SOURCES`. If no real sample exists, say so in a comment on the `Spec`, keep the parser conservative, and report the vendor as unverified.
- No cgo. Native libraries are loaded at run time with purego.
- Comments and help text: no em dashes, Oxford comma, digits for numbers, acronyms expanded on first use, no marketing words.
- Commit messages describe the change in the imperative and carry no trailers.

## Changelog

Add a line under `## Unreleased` in `CHANGELOG.md` for anything a user would notice: a new flag, a fixed bug, a new vendor. Skip it for internal refactors and typo fixes.

## Adding a vendor

1. Add the `device.Vendor` constant in `internal/device`.
2. Add `internal/provider/smi/<vendor>.go` with a `Spec`: the tool name and its usual absolute paths, `Args`, a `Parse` function that returns `[]device.Device`, and a `Hint` that says what to install. Prefer JSON or XML output, then `key: value`, then tables matched by unit rather than by column position.
3. Register the `Spec` in `smi.Vendors()`. Add it to the demo fleet in `internal/provider/sim` when it helps a screenshot.
4. Add a captured output file under `testdata/` and a line in `testdata/SOURCES`.
5. Add a parser test in `smi_test.go`.
6. Add the vendor to the table in `README.md`.

## Reporting a hardware run

Open an issue with the vendor, the tool version (`npu-smi -v`, `nvidia-smi --version`, and so on), the output of `siltide --once --json --vendors <vendor>`, and what looked wrong. Strip anything private from command lines first; `/api/snapshot` already does that, `--once --json` does not.
