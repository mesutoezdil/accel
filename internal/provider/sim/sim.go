// Package sim fakes a mixed accelerator fleet for `--demo`, so every view can
// be explored on any machine. Every name says "(simulated)".
package sim

import (
	"context"
	"math"
	"time"

	"github.com/mesutoezdil/accel/internal/device"
	"github.com/mesutoezdil/accel/internal/provider"
)

type model struct {
	vendor device.Vendor
	name   string
	count  int
	memGiB float64
	watts  float64 // 0: the tool reports no power
	temp   bool
	links  int // interconnect links per device
	procs  []proc
}

type proc struct {
	name, user, pod, ns, workload string
}

// Provider returns a demo provider with n NVIDIA-class devices plus one
// small group per other vendor. Device 3 of the NVIDIA group is a straggler
// on a narrow PCIe link; device n-1 is idle but allocated.
func Provider(n int) provider.Provider {
	train := proc{"python", "alice", "llama-70b-pretrain-0", "ml", "StatefulSet/llama-70b-pretrain"}
	serve := proc{"vllm", "svc", "chat-api-7d9f8b6c5-x2k9p", "inference", "Deployment/chat-api"}
	nb := proc{"python", "bob", "notebook-bob-0", "notebooks", "StatefulSet/notebook-bob"}
	models := []model{
		{device.NVIDIA, "NVIDIA H100 80GB HBM3", n, 80, 700, true, 18, []proc{train, serve}},
		{device.Ascend, "Ascend 910B3", 2, 64, 310, true, 7, []proc{train}},
		{device.Kunlunxin, "Kunlunxin P800 OAM", 1, 96, 400, true, 0, nil},
		{device.Cambricon, "Cambricon MLU370-X8", 1, 48, 250, true, 0, nil},
		{device.AMD, "AMD Instinct MI300X", 1, 192, 750, true, 7, []proc{serve}},
		{device.Neuron, "AWS Inferentia2", 1, 32, 0, false, 0, nil},
		{device.Apple, "Apple M4 Pro 20-core GPU", 1, 48, 0, false, 0, nil},
	}
	start := time.Now()
	return provider.Provider{
		Name: "sim", Label: "Simulated fleet",
		Detect: func() error { return nil },
		Read: func(context.Context) ([]device.Device, error) {
			var out []device.Device
			t := time.Since(start).Seconds()
			for vi, m := range models {
				for i := 0; i < m.count; i++ {
					load := 0.5 + 0.45*math.Sin(t/9+float64(vi*3+i))
					if m.vendor == device.NVIDIA { // a training job: pinned high, straggler aside
						load = 0.9 + 0.08*math.Sin(t/7+float64(i))
					}
					d := m.device(i, load, t)
					if m.vendor == device.NVIDIA && i == m.count-1 && m.count > 1 {
						d = m.device(i, 0.01, t) // idle but holding a notebook
						d.Procs = []device.Process{procOf(nb, 9000+i, d.Metrics[device.MemUsed], 0)}
					}
					out = append(out, d)
					if m.vendor == device.NVIDIA && i == 6 {
						out = append(out, m.partitions(d)...)
					}
				}
			}
			return out, nil
		},
	}
}

// device builds one device at the given load (0..1).
func (m model) device(i int, load, t float64) device.Device {
	d := device.New(m.vendor, i, m.name+" (simulated)", "", "")
	d.ID += "-sim"
	d.Source = "simulated"
	gib := m.memGiB * (1 << 30)
	throttle := 0
	if i == 3 { // the straggler: PCIe trained at x8, held back by power
		load, throttle = load*0.55, device.ThrottlePowerCap
	}
	d.Metrics[device.Util] = math.Round(load * 100)
	d.Metrics[device.MemUsed] = math.Round((0.2 + 0.7*load) * gib)
	d.Metrics[device.MemTotal] = gib
	if m.temp {
		d.Metrics[device.Temp] = math.Round(35 + 40*load)
	}
	if m.watts > 0 {
		d.Metrics[device.Power] = math.Round(m.watts * (0.15 + 0.8*load))
		d.Metrics[device.PowerCap] = m.watts
		d.Metrics[device.Throttle] = float64(throttle)
		d.Metrics[device.ClockCore] = math.Round(1000 + 900*load)
		d.Metrics[device.MemBandwidth] = math.Round(load * 70)
		d.Metrics[device.PCIeGen], d.Metrics[device.PCIeMaxGen] = 5, 5
		d.Metrics[device.PCIeWidth], d.Metrics[device.PCIeMaxWidth] = 16, 16
		d.Metrics[device.PCIeRx] = math.Round(load * 12e9)
		d.Metrics[device.PCIeTx] = math.Round(load * 4e9)
		d.Metrics[device.EccCorrected] = math.Floor(t / 90) // one every 90 s
		d.Metrics[device.EccUncorrected] = 0
		if i == 3 {
			d.Metrics[device.PCIeWidth] = 8
		}
	}
	if m.vendor == device.NVIDIA {
		d.Metrics[device.NUMANode] = float64(i / 4)
		d.Topology = map[int]string{}
		for peer := 0; peer < m.count; peer++ {
			switch {
			case peer == i:
			case peer/4 == i/4:
				d.Topology[peer] = "NV18"
			default:
				d.Topology[peer] = "SYS"
			}
		}
	}
	if m.links > 0 {
		d.Metrics[device.LinksTotal] = float64(m.links)
		d.Metrics[device.LinksActive] = float64(m.links)
		if i == 3 {
			d.Metrics[device.LinksActive] = float64(m.links - 1)
		}
	}
	for pi, p := range m.procs {
		share := 1 / float64(len(m.procs))
		d.Procs = append(d.Procs, procOf(p, 4000+10*i+pi, d.Metrics[device.MemUsed]*share, load*100*share))
	}
	return d
}

func procOf(p proc, pid int, mem, util float64) device.Process {
	pr := device.Process{
		PID: pid, Name: p.name, User: p.user, Command: p.name + " train.py --model " + p.workload,
		Pod: p.pod, Namespace: p.ns, Workload: p.workload, Container: "main",
		Started: time.Now().Add(-time.Duration(pid%7+1) * time.Hour),
		Metrics: device.Metrics{device.MemUsed: math.Round(mem), device.Util: math.Round(util)},
	}
	if p.name == "python" {
		pr.App = map[string]float64{"samples_per_s": math.Round(1500 + util*10), "nccl_gbps": math.Round(180 + util)}
	}
	return pr
}

// partitions splits one device into 2 MIG-style (Multi-Instance GPU) slices.
func (m model) partitions(parent device.Device) []device.Device {
	var out []device.Device
	for i, frac := range []float64{0.5, 0.25} {
		p := device.New(m.vendor, i, m.name+" 3g.40gb (simulated)", "", "")
		p.ID = parent.ID + "-part" + string(rune('0'+i))
		p.Parent, p.Source = parent.ID, "simulated"
		p.Metrics[device.MemTotal] = math.Round(parent.Metrics[device.MemTotal] * frac)
		p.Metrics[device.MemUsed] = math.Round(parent.Metrics[device.MemUsed] * frac)
		out = append(out, p)
	}
	parent.Metrics[device.Partitions] = float64(len(out))
	return out
}
