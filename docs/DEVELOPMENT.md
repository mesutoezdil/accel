# Developing siltide

Go 1.26 or newer, no cgo: NVML and the macOS frameworks load at run time through [purego](https://github.com/ebitengine/purego).

```sh
make build          # bin/siltide
make check          # lint, race tests, and cross builds
make demo           # the simulated fleet
make test-fake-nvml  # the NVML ABI test against a fake driver (Linux, or Docker elsewhere)
make shots           # fleet screenshots under assets/, from the simulated fleet
make shots-mac       # Apple silicon screenshots, captured on the Mac that runs it
```

CI runs gofmt, vet for Linux and macOS, race tests, golangci-lint, and cross builds.

## What a change needs

- **A test that fails without it.** The suite is the only thing standing
  between a parser and a machine nobody here can reach; a change that cannot
  be tested is usually a change that has not been thought through yet.
- **gofmt, vet for both platforms, and golangci-lint clean.** `make check`
  runs all of it.
- **Coverage that does not go down.** CI holds a floor and prints the
  per-function table.

## The layout

```text
main.go, cli.go        flags, the one-shot outputs, the manual page
internal/provider/     one package per vendor; smi/ holds the CLI parsers
internal/collect/      the engine: detect, collect, enrich, derive, record
internal/device/       the vendor-neutral model everything else speaks
internal/derive/       states, fleet totals, outliers, anomalies
internal/history/      the on-disk history and the time machine behind it
internal/kube/         pods, container ids, describe, logs, node resources
internal/query/        the filter language the interface, API and MCP share
internal/tui/          the interface
internal/server/       the API and the Prometheus endpoint
internal/mcp/          the Model Context Protocol server
scripts/               screenshots, the animation, the JSON schema, benchmarks
```

## Adding a vendor

Most vendors are a CLI that prints a table. `internal/provider/smi` holds one
file per vendor, each returning a `Spec` with the tool names to look for, a
hint for when it is missing, and a `Parse` over the output. A captured sample
goes in `testdata/`, a test holds the parser to it, and the fuzz target picks
it up as a seed automatically.

A vendor with a library rather than a CLI (NVML, IOKit) gets its own package
and loads it at run time through purego, so the binary still builds and runs
everywhere.

## Testing against a cluster

```sh
kind create cluster --name siltide
SILTIDE_TEST_CLUSTER=1 go test ./internal/kube/ -run Cluster -v
```

CI does this on every change that touches `internal/kube`.
