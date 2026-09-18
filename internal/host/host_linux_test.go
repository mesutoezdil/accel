package host

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, root, p, v string) {
	t.Helper()
	p = filepath.Join(root, p)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(v), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLinuxRates(t *testing.T) {
	root := t.TempDir()
	oldProc, oldSys := procRoot, sysRoot
	procRoot, sysRoot = filepath.Join(root, "proc"), filepath.Join(root, "sys")
	t.Cleanup(func() { procRoot, sysRoot = oldProc, oldSys })
	// 200 busy ticks out of 300 elapsed: 66.7% busy
	stat1 := "cpu  100 0 100 800 0 0 0 0\ncpu0 100 0 100 800 0 0 0 0\n"
	stat2 := "cpu  200 0 200 900 0 0 0 0\ncpu0 200 0 200 900 0 0 0 0\n"
	write(t, root, "proc/stat", stat1)
	write(t, root, "proc/meminfo", "MemTotal:       1000 kB\nMemAvailable:    400 kB\nSwapTotal:  100 kB\nSwapFree: 50 kB\n")
	write(t, root, "proc/loadavg", "1.5 1.0 0.5 1/100 123\n")
	write(t, root, "proc/uptime", "12345.6 1000.0\n")
	write(t, root, "proc/cpuinfo", "cpu MHz\t\t: 3000.0\n")
	write(t, root, "proc/mounts", "")
	write(t, root, "proc/diskstats", "   8       0 sda 10 0 1000 0 5 0 500 0 0 100 0\n   8       1 sda1 1 0 1 0 1 0 1 0 0 1 0\n")
	write(t, root, "sys/block/sda/size", "1")
	write(t, root, "proc/net/dev", "Inter-|   Receive                                                |  Transmit\n face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed\n  eth0: 1000 10 1 2 0 0 0 0 2000 20 3 4 0 0 0 0\n    lo: 5 5 0 0 0 0 0 0 5 5 0 0 0 0 0 0\n")
	write(t, root, "sys/class/net/eth0/operstate", "up\n")
	write(t, root, "sys/class/infiniband/mlx5_0/ports/1/state", "4: ACTIVE\n")
	write(t, root, "sys/class/infiniband/mlx5_0/ports/1/rate", "200 Gb/sec (4X HDR)\n")
	write(t, root, "sys/class/infiniband/mlx5_0/ports/1/counters/port_rcv_data", "100\n")
	write(t, root, "sys/class/infiniband/mlx5_0/ports/1/counters/port_xmit_data", "50\n")
	write(t, root, "sys/class/infiniband/mlx5_0/ports/1/counters/symbol_error", "2\n")

	s := New()
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return clock }
	first := s.Sample()
	if !first.CPU.Percent.Unknown() || first.Mem.Used != 600*1024 || first.CPU.Load1 != 1.5 || first.CPU.MHz != 3000 {
		t.Fatalf("first sample %+v", first)
	}
	write(t, root, "proc/stat", stat2)
	write(t, root, "proc/net/dev", "h\nh\n  eth0: 3000 30 1 2 0 0 0 0 4000 40 3 4 0 0 0 0\n")
	write(t, root, "sys/class/infiniband/mlx5_0/ports/1/counters/port_rcv_data", "200\n")
	write(t, root, "proc/diskstats", "   8       0 sda 20 0 3000 0 5 0 500 0 0 600 0\n")
	clock = clock.Add(2 * time.Second)
	second := s.Sample()
	if math.Abs(float64(second.CPU.Percent)-66.67) > 0.1 || len(second.CPU.PerCore) != 1 {
		t.Fatalf("cpu %v %v", second.CPU.Percent, second.CPU.PerCore)
	}
	if len(second.Nets) != 1 || second.Nets[0].RxBps != 1000 || second.Nets[0].TxPackets != 10 || second.Nets[0].RxDrops != 2 || !second.Nets[0].Up {
		t.Fatalf("net %+v", second.Nets)
	}
	if len(second.Disks) != 1 || second.Disks[0].ReadBps != 2000*512/2 || second.Disks[0].ReadIOPS != 5 || second.Disks[0].Busy != 25 {
		t.Fatalf("disk %+v", second.Disks)
	}
	if len(second.IB) != 1 || second.IB[0].RxBps != 200 || second.IB[0].State != "ACTIVE" || second.IB[0].Errors != 2 {
		t.Fatalf("ib %+v", second.IB)
	}
}

// TestLinuxSurvivesValuelessLines feeds the readers the shapes a hardened or
// half-mounted /proc produces: a key with nothing after it, an empty file,
// and a counter that reads blank. None of them may take the process down.
func TestLinuxSurvivesValuelessLines(t *testing.T) {
	root := t.TempDir()
	oldProc, oldSys := procRoot, sysRoot
	procRoot, sysRoot = filepath.Join(root, "proc"), filepath.Join(root, "sys")
	t.Cleanup(func() { procRoot, sysRoot = oldProc, oldSys })

	write(t, root, "proc/stat", "cpu  100 0 100 800 0 0 0 0\n")
	write(t, root, "proc/meminfo", "MemTotal:\nMemAvailable:    400 kB\nSwapTotal:\n")
	write(t, root, "proc/loadavg", "\n")
	write(t, root, "proc/uptime", "\n")
	write(t, root, "proc/cpuinfo", "")
	write(t, root, "proc/mounts", "")
	write(t, root, "proc/diskstats", "")
	write(t, root, "proc/net/dev", "")
	write(t, root, "sys/class/infiniband/mlx5_0/ports/1/state", "")
	write(t, root, "sys/class/infiniband/mlx5_0/ports/1/rate", "")
	write(t, root, "sys/class/infiniband/mlx5_0/ports/1/counters/port_rcv_data", "")
	write(t, root, "sys/class/infiniband/mlx5_0/ports/1/counters/port_xmit_data", " ")
	write(t, root, "sys/class/infiniband/mlx5_0/ports/1/counters/symbol_error", "")

	s := New()
	st := s.Sample()
	if st.Mem.Total.Unknown() || float64(st.Mem.Total) != 0 {
		t.Errorf("a MemTotal with no value should read as 0, got %v", st.Mem.Total)
	}
	if len(st.IB) != 1 || st.IB[0].Errors != 0 {
		t.Errorf("infiniband ports %+v", st.IB)
	}
	if !math.IsNaN(float64(st.CPU.Load1)) && st.CPU.Load1 != 0 {
		t.Errorf("load average from an empty file: %v", st.CPU.Load1)
	}
}
