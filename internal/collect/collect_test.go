package collect

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mesutoezdil/siltide/internal/config"
	"github.com/mesutoezdil/siltide/internal/device"
	"github.com/mesutoezdil/siltide/internal/history"
	"github.com/mesutoezdil/siltide/internal/provider"
)

func TestEngine(t *testing.T) {
	calls := 0
	ok := provider.Provider{
		Name: "ok", Detect: func() error { return nil },
		Read: func(context.Context) ([]device.Device, error) {
			calls++
			d := device.New(device.AMD, 0, "x", "", "")
			d.Metrics[device.Util] = 42
			d.Metrics[device.Temp] = 95
			d.Procs = []device.Process{{PID: 1, Name: "python", Metrics: device.Metrics{}}}
			return []device.Device{d}, nil
		},
	}
	gone := provider.Provider{Name: "gone", Detect: func() error { return errors.New("no tool") }}
	hist, _ := history.Open(history.Options{Keep: time.Hour, Resolution: time.Millisecond})
	e := New([]provider.Provider{gone, ok}, config.Default(), hist, false)
	e.Detect()
	e.Collect(context.Background())
	time.Sleep(2 * time.Millisecond)
	snap := e.Collect(context.Background())
	if len(snap.Devices) != 1 || calls != 2 {
		t.Fatalf("devices %d calls %d", len(snap.Devices), calls)
	}
	if snap.Providers[0].Active || snap.Providers[0].Error != "no tool" || !snap.Providers[1].Active {
		t.Fatalf("status %+v", snap.Providers)
	}
	d := snap.Devices[0]
	if d.State != device.StateActive || d.Health != 90 || len(snap.Alerts) != 1 || snap.Alerts[0].Kind != "thermal" {
		t.Fatalf("state %s health %d alerts %+v", d.State, d.Health, snap.Alerts)
	}
	if snap.Fleet.Active != 1 || snap.Fleet.Allocated != 1 {
		t.Fatalf("fleet %+v", snap.Fleet)
	}
	if h := hist.Recent("amd-0", device.Util, 10); len(h) != 2 || h[1] != 42 {
		t.Fatalf("history %v", h)
	}
}

func TestReconfigureAppliesANewConfig(t *testing.T) {
	hot := provider.Provider{
		Name: "hot", Detect: func() error { return nil },
		Read: func(context.Context) ([]device.Device, error) {
			d := device.New(device.AMD, 0, "x", "", "")
			d.Metrics[device.Util] = 42
			d.Metrics[device.Temp] = 70
			return []device.Device{d}, nil
		},
	}
	cfg := config.Default()
	cfg.Thresholds.TempWarn = 90
	e := New([]provider.Provider{hot}, cfg, nil, false)
	e.Detect()
	e.Collect(context.Background())
	snap := e.Collect(context.Background())
	if len(snap.Alerts) != 0 {
		t.Fatalf("70C should be quiet under a 90C threshold: %+v", snap.Alerts)
	}

	next := config.Default()
	next.Refresh = 5 * time.Second
	next.Thresholds.TempWarn = 60
	next.Carbon.GramsPerKWh = 250
	next.Cost.Currency = "€"
	e.Reconfigure(next)

	if e.Interval() != 5*time.Second {
		t.Errorf("interval %s, want 5s", e.Interval())
	}
	if e.Carbon() != 250 {
		t.Errorf("carbon %v, want 250", e.Carbon())
	}
	if e.Prices().Currency != "€" {
		t.Errorf("currency %q, want €", e.Prices().Currency)
	}
	e.Collect(context.Background())
	snap = e.Collect(context.Background()) // an alert needs two consecutive samples
	if len(snap.Alerts) != 1 || snap.Alerts[0].Kind != "thermal" {
		t.Fatalf("the reloaded 60C threshold did not fire: %+v", snap.Alerts)
	}
	if len(e.Events()) == 0 {
		t.Error("a reload should keep the event log")
	}
}
