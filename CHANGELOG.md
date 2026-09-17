# Changelog

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions follow [Semantic Versioning](https://semver.org/); until `1.0.0`, a minor bump can still break a flag or a config key.

## Unreleased

### Added
- 15 accelerator vendors: NVIDIA (NVML, no cgo), Apple silicon (ioreg, IOReport, SMC), AMD and Intel (sysfs, DRM fdinfo), and Huawei Ascend, AWS Neuron, Biren, Cambricon, Enflame, Hygon, Iluvatar, Kunlunxin, MetaX, Moore Threads, and VastAI through their CLI tools.
- Interactive terminal UI: 16 tabs, mouse support, a command bar with completion, a filter language, and 6 themes plus user-defined ones.
- History: a time machine on disk with scrubbing, zoom, and per-metric provenance.
- Kubernetes and Slurm process correlation, including the kubelet pod-resources API.
- An SSH provider that runs vendor CLIs on remote hosts without installing siltide there.
- Fleet mode: `--listen` serves `/api/snapshot`, `/api/summary`, `/api/events`, `/api/history`, and Prometheus `/metrics`; `--remote` attaches a TUI to another siltide.
- Cost and waste per workload, energy and carbon tracking, alert outputs (webhook, Slack, Alertmanager), anomaly detection, and topology-aware placement hints.
- `--record` and `--replay`, `--status`, `--export`, shell completions, and a man page.
- Packages: Linux and macOS binaries, deb and rpm, a container image, and a Homebrew tap.
