package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mesutoezdil/accel/internal/collect"
	"github.com/mesutoezdil/accel/internal/config"
	"github.com/mesutoezdil/accel/internal/history"
	"github.com/mesutoezdil/accel/internal/provider"
	"github.com/mesutoezdil/accel/internal/provider/sim"
)

func demoEngine(t *testing.T) *collect.Engine {
	t.Helper()
	hist, _ := history.Open(history.Options{Keep: time.Hour, Resolution: time.Millisecond})
	e := collect.New([]provider.Provider{sim.Provider(8)}, config.Default(), hist, true)
	e.Detect()
	e.Collect(context.Background())
	time.Sleep(2 * time.Millisecond)
	e.Collect(context.Background())
	return e
}

func TestEveryTabRenders(t *testing.T) {
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("default", nil)})
	m.width, m.height = 160, 50
	for i := range tabs {
		m.setTab(i)
		out := m.View()
		if strings.Contains(out, "%!") || strings.Count(out, "\n") < 3 {
			t.Errorf("tab %s rendered badly:\n%s", tabs[i].name, out)
		}
		if width(strings.Split(out, "\n")[0]) > 160 {
			t.Errorf("tab %s header wider than the screen", tabs[i].name)
		}
	}
	// narrow terminal must not panic
	m.width, m.height = 60, 15
	for i := range tabs {
		m.setTab(i)
		_ = m.View()
	}
}

func TestKeysAndCommands(t *testing.T) {
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("mono", nil), Keys: NewKeymap(map[string]string{"quit": "x"})})
	press := func(k string) {
		var msg tea.Msg
		if len(k) == 1 {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		} else {
			msg = tea.KeyMsg{Type: map[string]tea.KeyType{"enter": tea.KeyEnter, "esc": tea.KeyEsc, "tab": tea.KeyTab, "down": tea.KeyDown}[k]}
		}
		mm, _ := m.Update(msg)
		m = mm.(Model)
	}
	press("down")
	press("down")
	press("enter")
	if m.tab != tabDevices || m.sel != 2 {
		t.Fatalf("tab %d sel %d", m.tab, m.sel)
	}
	press(":")
	for _, c := range "proc" {
		press(string(c))
	}
	press("enter")
	if m.tab != tabProcesses {
		t.Fatalf("command :proc gave tab %d", m.tab)
	}
	press("/")
	for _, c := range "vllm" {
		press(string(c))
	}
	press("enter")
	for _, r := range m.processes() {
		if r.p.Name != "vllm" {
			t.Fatalf("filter leaked %s", r.p.Name)
		}
	}
	press(":")
	for _, c := range "theme nope" {
		press(string(c))
	}
	press("enter")
	if !strings.Contains(m.notice, "built in:") {
		t.Fatalf("notice %q", m.notice)
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if cmd == nil {
		t.Fatal("rebound quit key did nothing")
	}
}

func TestHistoryScrub(t *testing.T) {
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("default", nil)})
	m.setTab(tabHistory)
	m.scrub(-1)
	if m.cursor.IsZero() {
		t.Fatal("cursor should move off live")
	}
	if !strings.Contains(m.View(), "TIME MACHINE") {
		t.Fatal("header must show the time machine")
	}
	m.scrub(100)
	if !m.cursor.IsZero() {
		t.Fatal("scrubbing past the end returns to live")
	}
}

func TestPlain(t *testing.T) {
	e := demoEngine(t)
	out := Plain(e.Snapshot(), NewTheme("mono", nil), 120)
	if !strings.Contains(out, "DEMO") || !strings.Contains(out, "H100") || !strings.Contains(out, "WARNING") {
		t.Fatalf("plain:\n%s", out)
	}
}

func TestFilterLanguage(t *testing.T) {
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("default", nil)})
	m.setSearch("ns:inference")
	for _, d := range m.devices() {
		hit := false
		for _, p := range d.Procs {
			if p.Namespace == "inference" {
				hit = true
			}
		}
		if !hit {
			t.Fatalf("device %s passed ns:inference without a matching process", d.ID)
		}
	}
	m.setSearch("dev:3 vendor:nvidia")
	if devs := m.devices(); len(devs) != 1 || devs[0].Index != 3 {
		t.Fatalf("dev:3 -> %d devices", len(devs))
	}
	m.setTab(tabEvents)
	m.setSearch("sev:warning")
	for _, ev := range m.eventRows() {
		if ev.Severity != "warning" {
			t.Fatalf("sev filter leaked %s", ev.Severity)
		}
	}
}

func TestOverlaysAndTabs(t *testing.T) {
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("solarized", nil)})
	m.width, m.height = 140, 40
	m.setTab(tabKube)
	if len(m.pods()) == 0 {
		t.Fatal("demo must have pods")
	}
	m.describe()
	if m.overlay != overlayDescribe || !strings.Contains(m.View(), "Accelerators:") {
		t.Fatalf("describe overlay: %d", m.overlay)
	}
	m.logs()
	if m.overlay != overlayLogs {
		t.Fatal("logs overlay")
	}
	mm := m.overlayKey("esc")
	if mm.overlay != overlayNone {
		t.Fatal("esc closes the overlay")
	}
	m.startCompare([]string{"0", "1"})
	if !strings.Contains(m.View(), "Compare") {
		t.Fatal("compare view")
	}
	m.overlay = overlayNone
	for _, tab := range []int{tabNetwork, tabDashboard, tabHealth, tabNodes, tabWorkloads} {
		m.setTab(tab)
		if out := m.View(); strings.Contains(out, "%!") || len(out) < 50 {
			t.Fatalf("tab %d: %q", tab, out)
		}
	}
	m.setTab(tabOverview)
	m.sortBy(3)
	devs := m.devices()
	if len(devs) > 1 && devs[0].Metrics.Or("util", 0) < devs[1].Metrics.Or("util", 0) {
		t.Fatal("sort by util desc")
	}
}

func TestNodeCycleAndExport(t *testing.T) {
	e := demoEngine(t)
	m := New(e, Options{Theme: NewTheme("default", nil)})
	m.cycleNode(1)
	if m.node != "local" {
		t.Fatalf("node %q", m.node)
	}
	m.cycleNode(1)
	if m.node != "" {
		t.Fatalf("node %q", m.node)
	}
	var b strings.Builder
	n, err := ExportCSV(&b, e.History(), m.devices())
	if err != nil || n == 0 || !strings.HasPrefix(b.String(), "time,device,name,metric,value") {
		t.Fatalf("%d %v %q", n, err, b.String()[:min(40, len(b.String()))])
	}
	m.width, m.height = 160, 50
	m.setTab(tabLinks)
	if out := m.View(); !strings.Contains(out, "Topology") || !strings.Contains(out, "NV18") {
		t.Fatalf("links tab lacks the topology matrix")
	}
}
