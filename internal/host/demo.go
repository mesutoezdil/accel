package host

import (
	"math"
	"time"
)

// Demo returns a plausible 8-accelerator Linux host for `--demo`, so the
// demo never shows the machine it runs on. The values move a little with t.
func Demo(t time.Time, hostname string) Stats {
	w := math.Sin(float64(t.Unix()%60) / 9.5)
	var s Stats
	s.Time, s.Hostname, s.OS, s.Uptime = t, hostname, "linux/amd64", 87*3600+53*60
	s.CPU.Cores, s.CPU.Percent, s.CPU.MHz = 224, Float(38+6*w), 2900
	s.CPU.Load1, s.CPU.Load5, s.CPU.Load15 = Float(41.2+3*w), 38.9, 36.1
	for i := 0; i < s.CPU.Cores; i++ {
		s.CPU.PerCore = append(s.CPU.PerCore, Float(math.Abs(math.Mod(float64(i*37), 90)+8*w)))
	}
	tib := float64(1 << 40)
	s.Mem.Total, s.Mem.Used, s.Mem.Available = Float(2*tib), Float(1.12*tib), Float(0.88*tib)
	s.Mem.SwapTotal, s.Mem.SwapUsed = 0, 0
	s.Filesystems = []Filesystem{
		{Mount: "/", Type: "ext4", Total: 1.8 * tib, Used: 0.62 * tib},
		{Mount: "/data", Type: "xfs", Total: 30 * tib, Used: 21.4 * tib},
	}
	s.Disks = []Disk{
		{Name: "nvme0n1", ReadBps: 1.2e9 + 3e8*w, WriteBps: 4.1e8, ReadIOPS: 9800, WriteIOPS: 2100, Busy: 41 + 5*w},
		{Name: "nvme1n1", ReadBps: 9.7e8, WriteBps: 3.3e8, ReadIOPS: 7600, WriteIOPS: 1800, Busy: 33 + 4*w},
	}
	s.Nets = []Net{
		{Name: "bond0", Up: true, RxBps: 2.9e9 + 4e8*w, TxBps: 1.1e9, RxPackets: 410000, TxPackets: 230000},
		{Name: "eth0", Up: true, RxBps: 2.1e7, TxBps: 8.4e6, RxPackets: 3100, TxPackets: 1900},
	}
	for i := 0; i < 4; i++ {
		s.IB = append(s.IB, IBPort{Device: "mlx5_" + string(rune('0'+i)), Port: 1, State: "ACTIVE", Rate: "400 Gb/sec (4X NDR)", RxBps: 2.4e10 + 5e9*w, TxBps: 2.3e10})
	}
	return s
}
