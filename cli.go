package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/mesutoezdil/siltide/internal/collect"
	"github.com/mesutoezdil/siltide/internal/device"
	"github.com/mesutoezdil/siltide/internal/tui"
)

// flagNames are every long flag, for completions.
var flagNames = []string{
	"config", "interval", "demo", "demo-devices", "once", "json", "vendors", "listen", "service", "remote", "token",
	"gen-token", "no-history", "retention", "theme", "list-themes", "print-config", "debug", "log-file", "version",
	"record", "replay", "status", "export", "completion", "man",
}

// completion prints a shell completion script.
func completion(shell string) (string, error) {
	flags := "--" + strings.Join(flagNames, " --")
	switch shell {
	case "bash":
		return fmt.Sprintf(`# siltide bash completion: source this file or drop it in /etc/bash_completion.d
_siltide() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"
    case "$prev" in
        --theme) COMPREPLY=($(compgen -W "%s" -- "$cur")); return ;;
        --vendors) COMPREPLY=($(compgen -W "%s" -- "$cur")); return ;;
        --completion) COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur")); return ;;
        --config|--log-file|--record|--replay|--export) COMPREPLY=($(compgen -f -- "$cur")); return ;;
    esac
    COMPREPLY=($(compgen -W "%s" -- "$cur"))
}
complete -F _siltide siltide
`, strings.Join(tui.ThemeNames(), " "), strings.Join(names(), " "), flags), nil
	case "zsh":
		var lines []string
		for _, f := range flagNames {
			lines = append(lines, fmt.Sprintf("  '--%s'", f))
		}
		return fmt.Sprintf(`#compdef siltide
# siltide zsh completion: put this file in a directory on $fpath as _siltide
_arguments \
%s
`, strings.Join(lines, " \\\n")), nil
	case "fish":
		var b strings.Builder
		b.WriteString("# siltide fish completion: ~/.config/fish/completions/siltide.fish\n")
		for _, f := range flagNames {
			fmt.Fprintf(&b, "complete -c siltide -l %s\n", f)
		}
		fmt.Fprintf(&b, "complete -c siltide -l theme -xa '%s'\n", strings.Join(tui.ThemeNames(), " "))
		fmt.Fprintf(&b, "complete -c siltide -l vendors -xa '%s'\n", strings.Join(names(), " "))
		b.WriteString("complete -c siltide -l completion -xa 'bash zsh fish'\n")
		return b.String(), nil
	}
	return "", fmt.Errorf("unknown shell %q (bash, zsh, fish)", shell)
}

// manPage renders a roff manual page.
func manPage() string {
	var flags strings.Builder
	for _, f := range flagNames {
		fmt.Fprintf(&flags, ".TP\n.B \\-\\-%s\n", f)
		switch f {
		case "demo":
			flags.WriteString("Show a simulated mixed fleet instead of real hardware.\n")
		case "once":
			flags.WriteString("Print one snapshot and exit; exit code 3 when nothing was found.\n")
		case "json":
			flags.WriteString("JSON output: one snapshot with --once, otherwise a stream.\n")
		case "service":
			flags.WriteString("Run headless: collect, keep history, and serve the API and /metrics.\n")
		case "remote":
			flags.WriteString("Watch a remote siltide service (URL) instead of local hardware.\n")
		case "record":
			flags.WriteString("Append every snapshot as JSON to this file; replay it with --replay.\n")
		case "replay":
			flags.WriteString("Drive the interface from a recording instead of hardware.\n")
		case "status":
			flags.WriteString("Print a one-line summary for tmux, i3bar, or a prompt.\n")
		case "export":
			flags.WriteString("Write the on-disk history as CSV to this file and exit.\n")
		case "completion":
			flags.WriteString("Print a completion script for bash, zsh, or fish.\n")
		default:
			flags.WriteString("See siltide --help.\n")
		}
	}
	return fmt.Sprintf(`.TH SILTIDE 1 "%s" "siltide %s" "User Commands"
.SH NAME
siltide \- terminal monitor for GPUs, NPUs, and other AI accelerators
.SH SYNOPSIS
.B siltide
[\fIflags\fR]
.SH DESCRIPTION
siltide shows what every accelerator in the box (or the fleet) is doing right
now and, with its built-in time machine, what it did earlier. It reads NVIDIA,
AMD, Intel, Apple silicon, Huawei Ascend, AWS Inferentia and Trainium, Biren,
Cambricon, Enflame, Hygon, Iluvatar, Kunlunxin, MetaX, Moore Threads, and VastAI
devices without root, cgo, or a daemon, correlates processes with containers,
pods, Kubernetes workloads, and Slurm jobs, and exports everything as JSON and
Prometheus metrics.
.SH OPTIONS
%s
.SH FILES
.TP
.I ~/.config/siltide/config.yaml
Configuration; see examples/config.yaml. Unknown keys are rejected.
.TP
.I ~/.config/siltide/themes/*.yaml
User themes.
.TP
.I ~/.local/state/siltide/history/
History files, one per hour, plain text with a CRC per line.
.SH EXIT STATUS
0 on success, 1 on error, 2 on usage, 3 when --once found no accelerator.
.SH SEE ALSO
nvidia-smi(1), npu-smi(1)
`, time.Now().Format("2006-01-02"), version, flags.String())
}

// statusLine is the one-line summary for status bars.
func statusLine(s collect.Snapshot) string {
	if len(s.Devices) == 0 {
		return "siltide: no accelerators"
	}
	f := s.Fleet
	parts := []string{fmt.Sprintf("%d dev", f.Devices)}
	if !math.IsNaN(f.AvgUtil) {
		parts = append(parts, fmt.Sprintf("%.0f%% util", f.AvgUtil))
	}
	if f.MemTotal > 0 {
		parts = append(parts, fmt.Sprintf("%.0f%% mem", f.MemUsed/f.MemTotal*100))
	}
	if f.PowerW > 0 {
		parts = append(parts, fmt.Sprintf("%.0fW", f.PowerW))
	}
	if !math.IsNaN(f.MaxTemp) {
		parts = append(parts, fmt.Sprintf("%.0f°C", f.MaxTemp))
	}
	parts = append(parts, fmt.Sprintf("health %.0f", f.AvgHealth))
	if n := len(s.Alerts); n > 0 {
		parts = append(parts, fmt.Sprintf("▲%d", n))
	}
	worst := ""
	for _, d := range s.Devices {
		if d.State == device.StateDown {
			worst = " DOWN:" + d.Label()
			break
		}
	}
	return strings.Join(parts, " · ") + worst
}

// exportHistory writes the store as CSV to path.
func exportHistory(path string, eng *collect.Engine) error {
	h := eng.History()
	if h == nil {
		return fmt.Errorf("history is disabled")
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	var devs []device.Device
	for _, id := range h.IDs() {
		devs = append(devs, device.Device{ID: id})
	}
	n, err := tui.ExportCSV(f, h, devs)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "siltide: wrote %d rows to %s\n", n, path)
	return nil
}
