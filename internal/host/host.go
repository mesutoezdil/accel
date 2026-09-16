// Package host reads what the operating system knows about the machine:
// CPU, memory, filesystems, disks, network interfaces, and InfiniBand ports.
// Linux comes from `/proc` and `/sys`; macOS gets what `sysctl` and `vm_stat` give.
package host

import (
	"encoding/json"
	"math"
	"os"
	"runtime"
	"sync"
	"time"
)

// Float is a reading that may be unknown (NaN), written as null in JSON.
type Float float64

// MarshalJSON writes NaN as null.
func (f Float) MarshalJSON() ([]byte, error) {
	if math.IsNaN(float64(f)) {
		return []byte("null"), nil
	}
	return json.Marshal(float64(f))
}

// UnmarshalJSON reads null as NaN.
func (f *Float) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*f = Float(math.NaN())
		return nil
	}
	var v float64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*f = Float(v)
	return nil
}

// Unknown reports whether the value is NaN.
func (f Float) Unknown() bool { return math.IsNaN(float64(f)) }

// Stats is one host sample. NaN means unknown.
type Stats struct {
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	OS       string    `json:"os"`
	Uptime   float64   `json:"uptime_s"`

	CPU struct {
		Cores   int     `json:"cores"`
		Percent Float   `json:"percent"`  // busy share since the previous sample
		PerCore []Float `json:"per_core"` // same per core
		Load1   Float   `json:"load1"`
		Load5   Float   `json:"load5"`
		Load15  Float   `json:"load15"`
		MHz     Float   `json:"mhz"` // average current frequency
	} `json:"cpu"`

	Mem struct {
		Total     Float `json:"total"`
		Used      Float `json:"used"` // total minus available
		Available Float `json:"available"`
		SwapTotal Float `json:"swap_total"`
		SwapUsed  Float `json:"swap_used"`
	} `json:"mem"`

	Filesystems []Filesystem `json:"filesystems"`
	Disks       []Disk       `json:"disks"`
	Nets        []Net        `json:"nets"`
	IB          []IBPort     `json:"infiniband,omitempty"`
}

// Filesystem is a mounted filesystem.
type Filesystem struct {
	Mount string  `json:"mount"`
	Type  string  `json:"type"`
	Total float64 `json:"total"`
	Used  float64 `json:"used"`
}

// Disk is a block device with I/O rates.
type Disk struct {
	Name      string  `json:"name"`
	ReadBps   float64 `json:"read_bps"`
	WriteBps  float64 `json:"write_bps"`
	ReadIOPS  float64 `json:"read_iops"`
	WriteIOPS float64 `json:"write_iops"`
	Busy      float64 `json:"busy_percent"`
}

// Net is a network interface with rates since the previous sample.
type Net struct {
	Name      string  `json:"name"`
	Up        bool    `json:"up"`
	RxBps     float64 `json:"rx_bps"`
	TxBps     float64 `json:"tx_bps"`
	RxPackets float64 `json:"rx_pps"`
	TxPackets float64 `json:"tx_pps"`
	RxErrors  uint64  `json:"rx_errors"`
	TxErrors  uint64  `json:"tx_errors"`
	RxDrops   uint64  `json:"rx_drops"`
	TxDrops   uint64  `json:"tx_drops"`
}

// IBPort is an InfiniBand port with its counters.
type IBPort struct {
	Device string  `json:"device"`
	Port   int     `json:"port"`
	State  string  `json:"state"`
	Rate   string  `json:"rate"`
	RxBps  float64 `json:"rx_bps"`
	TxBps  float64 `json:"tx_bps"`
	Errors uint64  `json:"errors"` // symbol errors, link downed, and receive errors, summed
}

// Sampler keeps the previous counters to turn totals into rates.
type Sampler struct {
	mu   sync.Mutex
	prev raw
	at   time.Time
}

// raw holds counters from one read.
type raw struct {
	cpu    []cpuTimes // index 0 is the total
	disks  map[string]diskCounters
	nets   map[string]netCounters
	ib     map[string]ibCounters
	uptime float64
}

type cpuTimes struct{ user, nice, system, idle, iowait, irq, softirq, steal float64 }

func (c cpuTimes) total() float64 {
	return c.user + c.nice + c.system + c.idle + c.iowait + c.irq + c.softirq + c.steal
}
func (c cpuTimes) busy() float64 { return c.total() - c.idle - c.iowait }

type diskCounters struct{ rd, rdSectors, wr, wrSectors, ioMs float64 }
type netCounters struct {
	rxBytes, rxPackets, rxErr, txBytes, txPackets, txErr float64
	rxDrop, txDrop                                       float64 //nolint:unused // read on Linux
}
type ibCounters struct {
	rx, tx float64
	errs   float64 //nolint:unused // read on Linux
}

// New returns a sampler.
func New() *Sampler { return &Sampler{} }

// Sample reads the host. The first call has no rates (NaN).
func (s *Sampler) Sample() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	st := Stats{Time: now, OS: runtime.GOOS + "/" + runtime.GOARCH}
	st.Hostname, _ = os.Hostname()
	st.CPU.Cores = runtime.NumCPU()
	nan := Float(math.NaN())
	st.CPU.Percent, st.CPU.Load1, st.CPU.Load5, st.CPU.Load15, st.CPU.MHz = nan, nan, nan, nan, nan
	st.Mem.Total, st.Mem.Used, st.Mem.Available, st.Mem.SwapTotal, st.Mem.SwapUsed = nan, nan, nan, nan, nan

	cur := read(&st)
	dt := now.Sub(s.at).Seconds()
	if !s.at.IsZero() && dt > 0 {
		rates(&st, s.prev, cur, dt)
	}
	st.Uptime = cur.uptime
	s.prev, s.at = cur, now
	return st
}

// rates fills everything that needs 2 samples.
func rates(st *Stats, prev, cur raw, dt float64) {
	if len(prev.cpu) == len(cur.cpu) && len(cur.cpu) > 0 {
		pct := func(a, b cpuTimes) Float {
			if d := b.total() - a.total(); d > 0 {
				return Float((b.busy() - a.busy()) / d * 100)
			}
			return Float(math.NaN())
		}
		st.CPU.Percent = pct(prev.cpu[0], cur.cpu[0])
		for i := 1; i < len(cur.cpu); i++ {
			st.CPU.PerCore = append(st.CPU.PerCore, pct(prev.cpu[i], cur.cpu[i]))
		}
	}
	for name, c := range cur.disks {
		p, ok := prev.disks[name]
		if !ok {
			continue
		}
		st.Disks = append(st.Disks, Disk{
			Name: name, ReadBps: (c.rdSectors - p.rdSectors) * 512 / dt, WriteBps: (c.wrSectors - p.wrSectors) * 512 / dt,
			ReadIOPS: (c.rd - p.rd) / dt, WriteIOPS: (c.wr - p.wr) / dt, Busy: min((c.ioMs-p.ioMs)/(dt*1000)*100, 100),
		})
	}
	for i := range st.Nets {
		n := &st.Nets[i]
		c, p := cur.nets[n.Name], prev.nets[n.Name]
		if _, ok := prev.nets[n.Name]; !ok {
			continue
		}
		n.RxBps, n.TxBps = (c.rxBytes-p.rxBytes)/dt, (c.txBytes-p.txBytes)/dt
		n.RxPackets, n.TxPackets = (c.rxPackets-p.rxPackets)/dt, (c.txPackets-p.txPackets)/dt
	}
	for i := range st.IB {
		p := &st.IB[i]
		key := p.Device + "/" + itoa(p.Port)
		c, was := cur.ib[key]
		q, ok := prev.ib[key]
		if !was || !ok {
			continue
		}
		p.RxBps, p.TxBps = (c.rx-q.rx)*4/dt, (c.tx-q.tx)*4/dt // port counters are in 4-byte words
	}
	sortDisks(st)
}

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return itoa(i/10) + string(rune('0'+i%10))
}
