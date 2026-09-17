<p align="center">
  <img src="assets/wordmark.svg" alt="accel" width="560">
</p>

<p align="center">
  <a href="https://github.com/mesutoezdil/accel/actions/workflows/ci.yml"><img src="https://github.com/mesutoezdil/accel/actions/workflows/ci.yml/badge.svg" alt="ci"></a>
  <a href="https://github.com/mesutoezdil/accel/releases"><img src="https://img.shields.io/github/v/release/mesutoezdil/accel?include_prereleases&sort=semver" alt="release"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/mesutoezdil/accel" alt="go version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="license"></a>
</p>

accel is a terminal monitor for AI accelerators from 15 vendors: GPUs, NPUs, XPUs, MLUs, DCUs, GCUs, and Apple silicon. Utilization, memory, processes, power, thermals, links, and health per device; history on disk; pods and Slurm jobs next to processes; JSON and Prometheus for fleets.

<p align="center">
  <img src="assets/overview.png" alt="the Overview tab with a mixed fleet" width="100%">
</p>

## Contents

- [Supported accelerators](#supported-accelerators)
- [Install](#install)
- [Quick start](#quick-start)
- [Views](#views)
- [Keys](#keys)
- [Configuration](#configuration)
- [Fleets](#fleets)
- [Kubernetes and Slurm](#kubernetes-and-slurm)
- [API and Prometheus](#api-and-prometheus)
- [Alerts](#alerts)
- [History, recording, and export](#history-recording-and-export)
- [Themes and keymaps](#themes-and-keymaps)
- [Building and testing](#building-and-testing)
- [Contributing](#contributing)
- [License](#license)

## Supported accelerators

The vendor list follows the device plugins in [HAMi](https://github.com/Project-HAMi/HAMi/tree/master/pkg/device), plus Apple and Intel. Every vendor is auto-detected; `--vendors nvidia,ascend` limits the probe.

| Vendor | Devices | Source | Processes | Status |
|---|---|---|---|---|
| NVIDIA | GPUs, MIG slices | NVML loaded with `dlopen` (no cgo), Xid events, NVLink, ECC, row remap, PCIe AER via sysfs | yes, with `/proc` enrichment | verified on an H100 SXM (driver 570.211.01): every metric checked against `nvidia-smi`, idle and under a 2 GiB CUDA allocation. NVLink and MIG need a multi-GPU or MIG-enabled host, not tested |
| Apple | Apple silicon GPU | `ioreg`, IOReport (power, energy), SMC (temperature), AGX user clients (per-process GPU time) | yes | verified on an M4 Pro |
| AMD | Instinct, Radeon | sysfs (`/sys/class/drm`) and DRM `fdinfo` | yes | sysfs layout from kernel documentation; not yet run on hardware |
| Intel | Data Center GPU, Arc | sysfs and DRM `fdinfo` | yes | same as AMD |
| Huawei Ascend | NPUs | `npu-smi info` | yes | parser tested against captured output |
| AWS | Inferentia, Trainium | `neuron-ls`, `neuron-monitor` | no | parser tested against captured output |
| Cambricon | MLUs | `cnmon` | yes | parser tested against captured device rows |
| Enflame | GCUs | `efsmi` | no | parser tested against captured output (2 formats) |
| Hygon | DCUs | `hy-smi` (JSON) | no | parser tested against captured output |
| Iluvatar CoreX | GPUs | `ixsmi -q -x` (XML) | no | parser tested against captured output |
| Kunlunxin | XPUs | `xpu_smi` | no | parser tested against captured output |
| MetaX | GPUs | `mx-smi` | no | parser tested against captured output (2 formats) |
| Moore Threads | GPUs | `mthreads-gmi` | no | parser tested against captured output (2 formats) |
| Biren | GPUs | `brsmi` | no | flags from vendor documentation; no public sample, reported as unverified |
| VastAI | VA series | PCI sysfs | no | device presence only until the `vasmi` format is known |

Fixture sources: [`testdata/SOURCES`](internal/provider/smi/testdata/SOURCES). A value is measured or shown as `N/A`, never guessed; a tool whose output fails to parse shows its error in the Health tab.

Host metrics (CPU, memory, disks, network, InfiniBand) come from `/proc` and `/sys` on Linux and from `sysctl` and `vm_stat` on macOS.

## Install

Every release and every push to `main` (pre-release `vX.Y.Z-main.N`) publishes binaries, packages, and images.

**Release binary**

```sh
tag=$(curl -fsSL https://api.github.com/repos/mesutoezdil/accel/releases | grep -m1 '"tag_name"' | cut -d '"' -f4)
curl -fsSL -o accel "https://github.com/mesutoezdil/accel/releases/download/$tag/accel-linux-amd64"
chmod +x accel && sudo mv accel /usr/local/bin/
```

Every release so far is a pre-release, so GitHub's own `.../releases/latest` redirect 404s; the command above reads the newest tag from the API instead. Builds exist for `linux-amd64`, `linux-arm64`, `darwin-amd64`, and `darwin-arm64`. `checksums.txt` sits next to them. The macOS binaries are not notarized, so Gatekeeper quarantines a downloaded one; clear that before running it:

```sh
xattr -d com.apple.quarantine accel
```

**deb or rpm** (ships shell completions and the man page)

```sh
sudo dpkg -i accel_*_amd64.deb      # Debian, Ubuntu
sudo rpm -i accel-*.x86_64.rpm      # RHEL, Fedora, SUSE
```

**Homebrew** (macOS and Linux)

```sh
brew install mesutoezdil/tap/accel
```

After `brew tap mesutoezdil/tap`, plain `brew install accel` works too. The [tap](https://github.com/mesutoezdil/homebrew-tap) follows the newest release.

**Go**

```sh
go install github.com/mesutoezdil/accel@latest
```

**Container** (headless collector with the API and `/metrics` on port 9800)

```sh
docker run --rm -p 9800:9800 --gpus all --pid=host \
  -e NVIDIA_DRIVER_CAPABILITIES=utility \
  ghcr.io/mesutoezdil/accel:latest
```

`--pid=host` puts accel in the host's PID namespace; without it, `/proc` inside the container only shows the container's own processes, so accel finds the devices but not what is using them. Vendor CLIs must be visible inside the container too. A [systemd unit](deploy/systemd/accel.service) and a [Kubernetes DaemonSet](deploy/kubernetes/daemonset.yaml) are in `deploy/`.

## Quick start

```sh
accel                      # interactive terminal UI, vendors auto-detected
accel --demo               # explore every view with a simulated fleet
accel --once               # one snapshot on stdout (exit code 3 when nothing was found)
accel --once --json        # the same snapshot as JSON
accel --json               # a stream of JSON snapshots, one per refresh
accel --listen :9800       # the UI plus /api and /metrics
accel --service            # headless collector for fleets and Prometheus
accel --remote https://node:9800 --token ...   # the UI attached to a remote accel
accel --record run.jsonl   # record while running; accel --replay run.jsonl plays it back
accel --status             # one line for tmux, i3bar, or a shell prompt
```

`accel --status` prints, for the demo fleet:

```
17 dev · 59% util · 61% mem · 5035W · 74°C · health 98
```

## Views

16 tabs, switched by key or mouse. Tabs with nothing to show hide themselves.

| Key | Tab | What it shows |
|---|---|---|
| `1` | Overview | fleet summary, every device with utilization and memory bars, top processes, active alerts, spend and waste per hour |
| `2` | Devices | the device table plus a detail pane: sparklines, every metric with its unit and provenance, processes on the selected device |
| `3` | Processes | every process across devices with user, pod, job, memory, utilization, and runtime; sortable by any column |
| `4` | Memory | used, total, bandwidth, ECC counters, row remaps, retired pages |
| `5` | Power | draw, cap, energy since start, throttle reasons, power violations |
| `6` | Thermals | core and memory temperature, fan, thermal violations, warning threshold |
| `7` | Links | PCIe generation and width, NUMA node, RX and TX rates, replays, NVLink counts and errors, and the topology matrix |
| `8` | History | the time machine: every device as a sparkline over a zoomable window, a cursor to scrub, an inspector at the cursor, events up to the cursor |
| `9` | Events | Xids, link changes, throttle episodes, process starts and stops, alerts; filterable |
| `0` | Nodes | one row per node in a fleet with per-node totals; `ctrl+n` and `ctrl+p` cycle nodes in every other tab |
| `N` | Network | host interfaces and InfiniBand ports with rates, errors, and drops |
| `K` | Kubernetes | pods on this node with their devices, requests, and idle-allocated time; `d` describes, `l` shows logs |
| `W` | Workloads | Deployments, StatefulSets, Jobs, and Slurm jobs with devices held, efficiency, cost, kWh, and CO2 |
| `D` | Dashboard | fleet counters, a wide history chart, and the reliability table (Xid, ECC, remap, replays, violations, energy, link errors, health) |
| `H` | Health | the 0-100 score per device with every deduction explained, collector latency and errors, accel's own resource use |
| `?` | Help | keys, filter syntax, and the command list |

More tabs and a capture on Apple silicon: [SCREENSHOTS.md](SCREENSHOTS.md).

Values accel computes rather than reads (states, outliers, placement hints, anomalies, the health score) are labeled "(derived)".

## Keys

Defaults; rebind any action under `keys:` in the config.

| Keys | Action |
|---|---|
| `q`, `ctrl+c` | quit |
| `tab`, `right`, `]` and `shift+tab`, `left`, `[` | next and previous tab |
| `up`/`k`, `down`/`j`, `pgup`/`ctrl+u`, `pgdown`/`ctrl+d`, `home`/`g`, `end`/`G` | move in tables |
| `enter`, `esc` | open detail, go back |
| `p`, `space` | pause the display (collection continues) |
| `r` | refresh now |
| `/` | search; the filter language takes `dev:`, `user:`, `ns:`, `pod:`, `sev:`, `kind:`, and free text |
| `:` | command bar with Tab completion (`:theme dracula`, `:window 1h`, `:compare 0 3`, `:ns inference`, and more; `?` lists them) |
| `s`, `S` | sort by the next column, reverse the sort; a header click does the same |
| `,` and `.`, `<` and `>` | scrub history by 1 or 30 points |
| `n` | back to live |
| `m`, `M` | next and previous history metric |
| `+`/`=`, `-` | narrower and wider history window |
| `{`, `}` | previous and next device in History and Dashboard |
| `d`, `l`, `c`, `w` | describe pod, pod logs, next container, wrap long lines |
| `D` | jump to the Dashboard |
| `ctrl+n`, `ctrl+p` | next and previous node |
| `ctrl+e` | export the current table as CSV |

The mouse works everywhere: click tabs, rows, and headers; double-click a row for its detail; wheel to scroll. Hold Shift to select text.

## Configuration

`~/.config/accel/config.yaml` or `--config path`. Every key is optional; [`examples/config.yaml`](examples/config.yaml) lists them all with defaults, `accel --print-config` prints the effective result.

```yaml
refresh: 1s
history:
  keep: 24h
  resolution: 5s
thresholds:
  idle_util: 5
  busy_util: 80
  idle_after: 5m
  temp_warn: 85
cost:
  per_hour:
    "H100": 3.50
    "MI300X": 3.00
carbon:
  g_per_kwh: 400
```

State (history, the debug log) lives under `~/.local/state/accel`.

## Fleets

One accel per node serves; one accel shows them all.

On each node:

```sh
accel --gen-token                          # prints a token and its SHA-256 digest
accel --service --listen 0.0.0.0:9800 --token sha256:<digest> \
      --config /etc/accel/config.yaml      # with tls.cert and tls.key set
```

On your machine:

```yaml
nodes:
  - name: gpu-node-01
    url: https://gpu-node-01:9800
    token_file: ~/.config/accel/tokens/gpu-node-01
    ca: ~/.config/accel/ca.pem
  - name: hpc-07              # no accel installed there: vendor CLIs over ssh
    ssh: root@hpc-07
    key: ~/.ssh/id_ed25519
```

Remote and local devices share the same tables; the Nodes tab sums them. The `ssh` entry runs the vendor CLI on the remote host and parses it locally, so it works for the CLI vendors; NVIDIA, AMD, Intel, and Neuron need accel on the node.

Off loopback, `--listen` needs TLS and a token (`insecure: true` skips both). The config holds only the token digest. `/api/snapshot` strips command lines.

## Kubernetes and Slurm

Pods come from `/var/log/pods`; with in-cluster credentials or a kubeconfig (`kubernetes.kubeconfig`, `KUBECONFIG`, `~/.kube/config`) the API server adds owners, containers, and requests. The kubelet pod-resources socket shows which pod holds which device, even with no process running. Exec credential plugins are not run. The DaemonSet mounts everything needed.

On Slurm nodes the job comes from the process cgroup and `scontrol`; jobs appear in the Workloads tab.

## API and Prometheus

With `--listen` or `listen:` in the config:

| Path | Content |
|---|---|
| `GET /healthz` | `ok` |
| `GET /api/snapshot` | the latest snapshot: devices, processes, host, alerts, fleet totals |
| `GET /api/summary` | counts and totals only |
| `GET /api/events` | the event log |
| `GET /api/history?id=<device id>&n=600` | the last `n` points of utilization, memory, temperature, power, and core clock for one device |
| `GET /metrics` | Prometheus text format |

Auth: `Authorization: Bearer <token>`. Metrics are `accel_device_*` gauges (`util_percent`, `memory_used_bytes`, `power_watts`, `temperature_celsius`, `energy_joules_total`, `ecc_uncorrected_total`, `pcie_replays_total`, `links_active`, `health_score`, and more) with `node`, `vendor`, `index`, `id`, and `name` labels, plus `accel_host_*`. Unreported metrics are absent.

`--json` streams one snapshot per refresh; `--record` writes the same format and `--replay` reads it.

## Alerts

Built in: temperature at `temp_warn`, thermal and power throttling, hardware slowdown, outliers, idle-allocated devices, row remaps, a narrowed PCIe link, links down. Each needs 2 consecutive samples. Your own:

```yaml
alerts:
  min_severity: warning
  resend: 1h
  webhook: https://hooks.example.com/accel        # POST {"host", "alerts": [...]}
  slack: https://hooks.slack.com/services/...
  alertmanager: http://alertmanager:9093
  rules:
    - name: wasted
      when: "util < 10"          # <metric> <op> <value>; memory values take K, M, G, T
      for: 10m
      on: allocated              # "", allocated, or idle
      severity: warning
    - when: "health < 50"
      severity: critical
```

## History, recording, and export

History is plain-text hourly files, capped by `history.max_disk_mb`, reloaded on start. `--retention 168h` overrides `history.keep`; `accel --export history.csv` writes CSV. `--record` and `--replay` let you hand an incident to someone else.

## Themes and keymaps

Themes: `amber`, `default`, `dracula`, `ice`, `mono`, `solarized`. Your own goes in `~/.config/accel/themes/<name>.yaml` (`extends` a built-in, override any of the 19 roles; see [`examples/themes/corp.yaml`](examples/themes/corp.yaml)). `colors:` overrides single roles; `transparent: true` keeps the terminal background.

```yaml
keys:
  quit: "q,ctrl+c"
  command: ";"
```

## Building and testing

Go 1.26 or newer, no cgo: NVML and the macOS frameworks load at run time through [purego](https://github.com/ebitengine/purego).

```sh
make build        # bin/accel
make check        # lint, race tests, and cross builds
make demo         # the simulated fleet
make test-fake-nvml   # the NVML ABI test against a fake driver (Linux, or Docker elsewhere)
make shots        # the fleet screenshots under assets/, rendered from the simulated fleet (synthetic numbers)
make shots-mac    # the Apple silicon screenshots, captured on the Mac that runs it
```

CI runs gofmt, vet for Linux and macOS, race tests, golangci-lint, and cross builds. The fake NVML test checks the ABI structs against a C library without hardware.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to add a vendor and what a change needs before it merges. Security reports: [SECURITY.md](SECURITY.md). The roadmap is the [issue list](https://github.com/mesutoezdil/accel/issues); changes are in [CHANGELOG.md](CHANGELOG.md).

## License

Apache-2.0. See [LICENSE](LICENSE).
