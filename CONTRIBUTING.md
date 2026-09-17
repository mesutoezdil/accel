# Contributing

Thanks for helping. Issues and pull requests are welcome; hardware reports most of all, since most vendors are tested only against captured tool output.

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

## Adding a vendor

1. Add the `device.Vendor` constant in `internal/device`.
2. Add `internal/provider/smi/<vendor>.go` with a `Spec`: the tool name and its usual absolute paths, `Args`, a `Parse` function that returns `[]device.Device`, and a `Hint` that says what to install. Prefer JSON or XML output, then `key: value`, then tables matched by unit rather than by column position.
3. Register the `Spec` in `smi.Vendors()`. Add it to the demo fleet in `internal/provider/sim` when it helps a screenshot.
4. Add a captured output file under `testdata/` and a line in `testdata/SOURCES`.
5. Add a parser test in `smi_test.go`.
6. Add the vendor to the table in `README.md` and tick it in `TODO.md`.

## Reporting a hardware run

Open an issue with the vendor, the tool version (`npu-smi -v`, `nvidia-smi --version`, and so on), the output of `accel --once --json --vendors <vendor>`, and what looked wrong. Strip anything private from command lines first; `/api/snapshot` already does that, `--once --json` does not.
