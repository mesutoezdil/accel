# Performance baselines

The site publishes what siltide costs to run. This page is how those numbers
were produced, so they can be produced again on a machine that is not the one
they came from. Nothing here is estimated: every row is the output of a
command in this repository.

## The machine

| | |
| --- | --- |
| Hardware | Apple M5, 10 cores, 24 GB |
| System | macOS 26.6.2 |
| Go | 1.26.6 |
| siltide | v0.0.37-main.1-4-g16fd798 |
| Accelerators present | one Apple GPU |

A Linux host with eight NVIDIA cards will not reproduce these numbers. The
startup figure in particular moves with how many vendors are installed,
because most of it is probing them.

## Startup and memory

```sh
make build
scripts/perf/startup.sh 20     # median of 20 runs
scripts/perf/memory.sh 30      # the interface, sampled every 2s
scripts/perf/memory.sh 30 --service   # the headless collector
```

| Measurement | Result | What it covers |
| --- | ---: | --- |
| First rendered snapshot, simulated fleet | 0.03 s | process start, one collection, one render, no hardware |
| First rendered snapshot, this machine | 0.52 s | the same, plus probing fifteen vendors |
| of which vendor detection | 0.43 s | what `siltide --diagnose` reports on its last line |
| Resident memory, interface | 19 MB | 17 devices in view, history buffers filled |
| Resident memory, headless collector | 25 MB | `--service`, serving the API and /metrics |

Both memory scripts sample every two seconds and report the median, because
the first reading is taken before the history buffers have filled and
flatters the number.

## The parts, measured separately

```sh
make bench
```

| Benchmark | Result | What it is |
| --- | ---: | --- |
| `BenchmarkFilter` | 11 µs | one keystroke in the filter bar: parse the query, match every device and process in view |
| `BenchmarkRender` | 0.38 ms | redrawing the Overview at 200 by 60 cells |
| `BenchmarkRenderDashboard` | 0.51 ms | the same for the Dashboard |
| `BenchmarkCollect/64` | 0.60 ms | one collection pass over 64 devices, excluding what the vendor tool itself costs |
| `BenchmarkRecord` | 7.5 µs | writing one history point to disk |

## What is not measured here

- **No comparison against another tool.** Running one properly means the same
  fleet, the same window and the same definition of "started", and none of
  that has been done. A number without that is an advertisement.
- **Vendor tool cost.** `BenchmarkCollect` measures siltide's work over
  already-parsed output. On a real NVIDIA host most of a pass is NVML.
- **Anything about a large fleet.** These are one machine and a simulated
  seventeen-device fleet. Numbers from a real rack are welcome as an issue.
