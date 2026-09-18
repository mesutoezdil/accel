# Using siltide

```sh
siltide                      # interactive terminal UI, vendors auto-detected
siltide --demo               # explore every view with a simulated fleet
siltide --once               # one snapshot on stdout (exit code 3 when nothing was found)
siltide --once --json        # the same snapshot as JSON
siltide --json               # a stream of JSON snapshots, one per refresh
siltide --listen :9800       # the UI plus /api and /metrics
siltide --service            # headless collector for fleets and Prometheus
siltide --remote https://node:9800 --token ...   # the UI attached to a remote siltide
siltide --record run.jsonl   # record while running, siltide --replay run.jsonl plays it back later
siltide --status             # one line for tmux, i3bar, or a shell prompt
```

`siltide --status` prints, for the demo fleet: `17 dev · 59% util · 61% mem · 5035W · 74°C · health 98`.

## In the interface

`?` lists every tab, key and filter at any time. The parts worth knowing
before that:

- **Tabs** are on the number and letter keys, and `:` takes their names too.
- **`/` filters** everything in view. The language takes free words, `key:value`
  pairs (`dev:3`, `ns:ml`, `vendor:nvidia`), comparisons (`util>80`,
  `temp>=70`, `mem<50`, `mem_used>8G`), and `!` to negate any of them.
- **`:` is the command bar.** It lists what it accepts as you type, with the
  values each command takes: theme names, metrics, windows, the columns of
  the tab in view, the namespaces in the cluster, and your saved views.
- **Bookmarks**: `:bookmark save incident` records the tab, filter, node and
  namespace; `:bookmark incident` returns to it. The last view is restored on
  the next run by itself.
- **History**: `,` and `.` scrub back and forward, `n` returns to now, `m`
  changes the metric, `+` and `-` the window.
- **`:reload`** re-reads the config file without restarting, and **`:log`**
  shows siltide's own log when `--debug` is on.

A view can also be opened straight from the command line:

```sh
siltide --tab processes --filter "util>80"
siltide --bookmark incident
```

`~/.config/siltide/config.yaml` or `--config path`, every key documented in [`examples/config.yaml`](../examples/config.yaml). `siltide --print-config` shows what is active. `?` in the terminal lists every tab, key, and filter.

- **Fleets**: `siltide --service --listen 0.0.0.0:9800 --gen-token` on each node, then list them under `nodes:` on the machine you watch from, or reach a vendor CLI over `ssh` without installing siltide there. Off loopback, `--listen` needs TLS and a token unless `insecure: true`.
- **Kubernetes and Slurm**: pods and Slurm jobs show up next to processes on their own, from `/var/log/pods`, the kubelet pod-resources socket, or `scontrol`. The DaemonSet mounts what it needs.
- **API and Prometheus**: `--listen` serves `GET /api/snapshot`, `/api/summary`, `/api/events`, `/api/history?id=<device id>&n=600`, and Prometheus `/metrics` (`siltide_device_*` and `siltide_host_*` gauges), bearer-token authenticated.
- **Alerts**: built-in ones for temperature, throttling, outliers, idle-allocated devices, row remaps, and link degradation, plus your own rules under `alerts:`.
- **History**: plain-text hourly files, `--retention` to override how long, `--export` to CSV, `--record`/`--replay` to hand off an incident.
- **Themes**: fifteen built in, including `paper` for a light terminal, or your own in `~/.config/siltide/themes/` (see [`../examples/themes/corp.yaml`](../examples/themes/corp.yaml)). `siltide --list-themes` prints them.

## Answering an agent

`--mcp-stdio` and `--mcp-http` serve the same snapshot over the Model Context
Protocol. See [MCP.md](MCP.md).
