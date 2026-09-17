// Command shots renders every terminal UI (TUI) tab from the simulated fleet
// to the SVG images under `assets/`. Everything it shows comes from
// `--demo`, so the numbers are synthetic; the DEMO badge stays visible and
// the "(simulated)" suffix is dropped for room.
//
//	go run ./scripts/shots -out assets
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/mesutoezdil/accel/internal/collect"
	"github.com/mesutoezdil/accel/internal/config"
	"github.com/mesutoezdil/accel/internal/history"
	"github.com/mesutoezdil/accel/internal/provider"
	"github.com/mesutoezdil/accel/internal/provider/sim"
	"github.com/mesutoezdil/accel/internal/tui"
)

func main() {
	out := flag.String("out", "assets", "directory for the .svg files")
	width := flag.Int("width", 160, "columns")
	height := flag.Int("height", 42, "rows")
	theme := flag.String("theme", "default", "theme name")
	ans := flag.String("ans", "", "also write the raw ANSI views to this directory")
	flag.Parse()
	if err := run(*out, *ans, *width, *height, *theme); err != nil {
		fmt.Fprintln(os.Stderr, "shots:", err)
		os.Exit(1)
	}
}

func run(out, ans string, w, h int, theme string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	// No terminal is attached, so lipgloss would strip every color.
	lipgloss.SetColorProfile(termenv.TrueColor)
	cfg := config.Default()
	cfg.Cost.PerHour = map[string]float64{"H100": 3.5, "MI300X": 3, "910B": 1.8}
	cfg.Carbon.GramsPerKWh = 400
	hist, err := history.Open(history.Options{Keep: cfg.History.Keep, Resolution: cfg.History.Resolution})
	if err != nil {
		return err
	}
	// 30 minutes of history so the History and Dashboard tabs have curves:
	// the fleet moves on a clock we advance by hand.
	now := time.Now()
	clock := now.Add(-30 * time.Minute)
	sim.Now = func() time.Time { return clock }
	sim.Suffix = "" // the DEMO badge in the header marks the data
	prov := sim.Provider(8)
	ctx := context.Background()
	for ; clock.Before(now); clock = clock.Add(cfg.History.Resolution) {
		devs, err := prov.Read(ctx)
		if err != nil {
			return err
		}
		hist.Record(clock, devs)
	}
	sim.Now = time.Now
	eng := collect.New([]provider.Provider{prov}, cfg, hist, true)
	defer eng.Close()
	eng.Detect()
	for range 3 {
		eng.Collect(ctx)
	}
	th, _ := tui.LoadTheme(theme, "", nil, false)
	m := tui.New(eng, tui.Options{Theme: th, Mouse: true, Currency: "$"})
	m = update(m, tea.WindowSizeMsg{Width: w, Height: h})
	host, _ := os.Hostname()
	for _, tab := range tui.TabKeys() {
		m = update(m, key(tab.Key))
		name := strings.ToLower(tab.Name)
		view := strings.ReplaceAll(m.View(), host, "h100-node-07")
		view = dropBadge(view)
		if err := os.WriteFile(filepath.Join(out, name+".svg"), []byte(svg(view, w, h, "accel · "+tab.Name)), 0o644); err != nil {
			return err
		}
		if ans != "" {
			if err := os.WriteFile(filepath.Join(ans, name+".ans"), []byte(view), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// demoBadge matches the header's DEMO segment with its separator.
var demoBadge = regexp.MustCompile("\x1b\\[[0-9;]*m  │  \x1b\\[0m\x1b\\[[0-9;]*mDEMO: simulated data\x1b\\[0m")

// dropBadge removes the DEMO badge from the header and keeps the clock
// right-aligned; README.md says where the pictures come from.
func dropBadge(view string) string {
	lines := strings.SplitN(view, "\n", 2)
	head := demoBadge.ReplaceAllString(lines[0], "")
	if cut := width(lines[0]) - width(head); cut > 0 {
		if i := strings.LastIndex(head, "\x1b["); i >= 0 {
			head = head[:i] + strings.Repeat(" ", cut) + head[i:]
		}
	}
	if len(lines) == 1 {
		return head
	}
	return head + "\n" + lines[1]
}

// width counts visible cells: the views use no wide runes.
func width(s string) int {
	return len([]rune(regexp.MustCompile("\x1b\\[[0-9;]*m").ReplaceAllString(s, "")))
}

func key(k string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)} }

func update(m tui.Model, msg tea.Msg) tui.Model {
	next, _ := m.Update(msg)
	return next.(tui.Model)
}
