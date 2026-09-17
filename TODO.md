# accel Roadmap

Tick a box when the item is built, tested, and lint-clean. Commit and push
only when the owner says so.

## Vendors (15, every one in HAMi plus Apple and Intel)
- [x] NVIDIA (NVIDIA Management Library (NVML) via purego, no cgo)
- [x] Apple silicon (ioreg)
- [x] AMD (sysfs)
- [x] Intel (sysfs)
- [x] Huawei Ascend (npu-smi)
- [x] AWS Inferentia/Trainium (neuron-ls, neuron-monitor)
- [x] Biren (brsmi, unverified on hardware)
- [x] Cambricon (cnmon)
- [x] Enflame (efsmi)
- [x] Hygon (hy-smi)
- [x] Iluvatar (ixsmi)
- [x] Kunlunxin (xpu_smi)
- [x] MetaX (mx-smi)
- [x] Moore Threads (mthreads-gmi)
- [x] VastAI (PCI only, vasmi output format unknown)

## Core
- [x] Vendor-neutral device model, missing metric is N/A, never 0
- [x] Collector with per-provider timeouts and parallel reads
- [x] Persistent history (time machine) with hourly plain-text files
- [x] Health score with itemized reasons
- [x] Events and active alerts
- [x] Derived: states, outliers, idle-allocated, fleet totals, unused equivalents
- [x] Process enrichment: user, command, container, pod UID (cgroups)
- [x] Kubernetes: /var/log/pods and in-cluster API, workload inference
- [x] YAML config with thresholds, nodes, theme, colors, keys
- [x] Prometheus /metrics, JSON API, bearer token
- [x] Remote nodes as providers (fleet in one screen), `--remote`
- [x] `--once`, `--json` snapshot and `--json` stream, `--gen-token`, `--version`
- [x] Demo fleet covering every view

## Parity: host and network
- [x] Host CPU (total, per core, load, frequency), memory, swap
- [x] Filesystems and disk I/O rates
- [x] Network interfaces: RX/TX rates, packets, errors, drops
- [x] InfiniBand port counters
- [x] Nodes tab shows host metrics; Network tab

## Parity: NVIDIA depth
- [x] Xid events through the NVML event API, with a severity catalog
- [x] Row remapping, retired pages, PCIe replays, violation counters
- [x] Energy since driver load, P-state, memory temperature, slowdown/shutdown thresholds
- [x] NVLink per link: state, version, peer, throughput, error counters
- [x] Per-process SM/memory utilization
- [x] Multi-Instance GPU (MIG) instance processes and utilization where NVML allows

## Parity: security
- [x] TLS for `--listen`, mTLS optional
- [x] Token stored as SHA-256 digest, constant-time check
- [x] Refuse to bind outside loopback without TLS and a token
- [x] Remote client: CA file, token file

## Parity: Kubernetes
- [x] Pod describe (containers, requests/limits, conditions, labels, events)
- [x] Container logs (API, or the pod log files on the node without credentials)
- [x] kubeconfig support outside the cluster (token, client certificates)
- [x] `:ns` and pod-name commands

## Parity: UI
- [x] Dashboard tab: stat tiles and one chart per metric with a line per device
- [x] Health tab with reasons, collector status, and accel's own resource use
- [x] History zoom (+ -), device pick ([ ]), metric pick (m/M), now (n)
- [x] Column header click to sort, double-click for detail
- [x] Filter language: `gpu:0 user:alice ns:ml sev:critical kind:xid`
- [x] Themes with `extends` and 19 colour roles; `--list-themes`; `transparent`
- [x] `--print-config`, `--debug` and `--log-file`, exit code 3 when nothing found
- [x] Layouts from 60x15 up

## Parity: quality and release
- [x] NVML struct layout tests
- [x] Fake NVML C library and end-to-end ABI test (Linux)
- [x] Benchmarks for collection, publish, and render
- [x] Versioned JSON schema ("accel.snapshot/v1")
- [x] History: CRC per line, disk cap, single-writer lock, torn tail tolerance
- [x] Unknown config keys rejected with line numbers
- [x] Prerelease on every push to main; deb, rpm, and Homebrew via goreleaser
- [x] Container image

## Beyond parity
- [x] Kubelet pod-resources API: allocation without running processes
- [x] Slurm: job to device mapping, per-job efficiency
- [x] SSH provider: read vendor tools on remote hosts without installing anything
- [x] Cost view: per-device price, spend, and waste per workload
- [x] Alert outputs: generic webhook, Slack, Alertmanager
- [x] Compare two devices or two moments side by side
- [x] Per-vendor depth beyond utilization (processes for AMD, Ascend, Cambricon)
- [x] README with screenshots rendered from `--demo` (`make shots`), logo, contributing, security, and issue templates
- [x] Homebrew tap: `mesutoezdil/homebrew-tap` renders its formula from the newest release on a schedule, so no cross-repository token is needed
- [ ] eBPF kernel-launch visibility (long term)

## Remaining gaps against the inspiration
- [x] Apple: IOReport power, energy, and frequency, System Management Controller (SMC) temperature, per-process GPU time (purego, no root)
- [x] NVIDIA: NVLink topology matrix (common ancestor), non-uniform memory access (NUMA) node, PCIe Advanced Error Reporting (AER) counters
- [x] Health: Xid, NVLink errors, remap failure, and recovery-action deductions
- [x] Events: process start/stop, NVLink up/down, throttle start/end debounced
- [x] History keeps throttle reasons (OR-ed) so bursts survive downsampling
- [x] Per-metric provenance (nvml, ioreg, npu-smi, sysfs, derived) shown in detail and JSON
- [x] Processes: runtime since start, memory share of the device
- [x] Dashboard reliability table: Xid, error-correcting code (ECC) errors, remaps, replays, violations, energy
- [x] Wheel scrolls the detail pane under the pointer; Shift selects text

## Beyond, round two
- [x] Data Center GPU Manager (DCGM) profiling through `dcgmi` (SM active, occupancy, tensor active, DRAM active): is the GPU really computing
- [x] Record and replay: `--record file` writes the JSON stream, `--replay file` drives the terminal UI (TUI) from it (share an incident)
- [x] User alert rules in the config: `util < 10 for 10m on allocated` and friends
- [x] Energy and carbon per workload: kWh from the energy counter, gCO2/kWh from the config
- [x] `--status` one-line output for tmux, i3bar, and prompts
- [x] `--export` history to CSV for spreadsheets and notebooks
- [x] Anomaly detection on history: utilization collapse, memory leak slope, temperature creep
- [x] Topology-aware placement hints: which free devices share an NVLink or NUMA domain
- [x] Shell completions (bash, zsh, fish) and a man page in the packages
- [x] Node switching inside the TUI with live per-node dashboards
- [x] Application metrics when workloads expose them (training throughput, NCCL bandwidth)

## Notes
- The fake NVML ABI test needs a C compiler on Linux; CI runs it on
  `ubuntu-latest`. Locally, `scripts/test-fake-nvml.sh` runs it (through
  Docker off Linux).
- Vendor CLIs over SSH: Ascend, Cambricon, Enflame, Hygon, Iluvatar,
  Kunlunxin, MetaX, Moore Threads, Biren. NVIDIA, AMD, Intel, and Neuron need
  accel on the node (`--listen`).
- Kubernetes exec credential plugins are not run; such clusters fall back to
  the log directory.

## Validation on real hardware
- [ ] NVIDIA host: every metric, Xids, NVLink, MIG
- [ ] A Kubernetes cluster: pod correlation, pod-resources, describe, logs
- [ ] Each CLI vendor on its hardware (Biren and VastAI output formats first)
- [x] Apple: SMC temperature and power
