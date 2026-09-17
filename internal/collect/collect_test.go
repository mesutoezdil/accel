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
