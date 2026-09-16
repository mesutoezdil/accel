//go:build darwin

package host

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// macOS has no `/proc`; `sysctl` and `vm_stat` answer what the Nodes tab shows.
// ponytail: per-core CPU and disk rates need host_statistics through
// purego; add when someone monitors a Mac fleet.
func read(st *Stats) raw {
	r := raw{disks: map[string]diskCounters{}, nets: map[string]netCounters{}, ib: map[string]ibCounters{}}
	if out, err := exec.Command("sysctl", "-n", "vm.loadavg", "hw.memsize", "kern.boottime").Output(); err == nil {
		ls := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(ls) >= 1 {
			fs := strings.Fields(strings.Trim(ls[0], "{} "))
			if len(fs) >= 3 {
				st.CPU.Load1, st.CPU.Load5, st.CPU.Load15 = Float(f(fs[0])), Float(f(fs[1])), Float(f(fs[2]))
			}
		}
		if len(ls) >= 2 {
			st.Mem.Total = Float(f(ls[1]))
		}
		if len(ls) >= 3 {
			if _, sec, ok := strings.Cut(ls[2], "sec = "); ok {
				n, _ := strconv.ParseFloat(strings.TrimSpace(strings.Split(sec, ",")[0]), 64)
				r.uptime = uptimeSince(n)
			}
		}
	}
	if out, err := exec.Command("vm_stat").Output(); err == nil {
		page, free := 4096.0, 0.0
		for _, l := range strings.Split(string(out), "\n") {
			switch {
			case strings.HasPrefix(l, "Mach Virtual Memory Statistics"):
				if _, v, ok := strings.Cut(l, "page size of "); ok {
					page = f(strings.Fields(v)[0])
				}
			case strings.HasPrefix(l, "Pages free:"), strings.HasPrefix(l, "Pages inactive:"), strings.HasPrefix(l, "Pages speculative:"):
				_, v, _ := strings.Cut(l, ":")
				free += f(strings.TrimSuffix(strings.TrimSpace(v), "."))
			}
		}
		if st.Mem.Total > 0 {
			st.Mem.Available = Float(free * page)
			st.Mem.Used = st.Mem.Total - st.Mem.Available
		}
	}
	if out, err := exec.Command("sysctl", "-n", "vm.swapusage").Output(); err == nil {
		s := string(out)
		if _, t, ok := strings.Cut(s, "total = "); ok {
			st.Mem.SwapTotal = Float(mb(strings.Fields(t)[0]))
		}
		if _, u, ok := strings.Cut(s, "used = "); ok {
			st.Mem.SwapUsed = Float(mb(strings.Fields(u)[0]))
		}
	}
	for _, mount := range []string{"/", "/System/Volumes/Data"} {
		var sf syscall.Statfs_t
		if syscall.Statfs(mount, &sf) == nil && sf.Blocks > 0 {
			bs := float64(sf.Bsize)
			st.Filesystems = append(st.Filesystems, Filesystem{Mount: mount, Type: "apfs", Total: float64(sf.Blocks) * bs, Used: float64(sf.Blocks-sf.Bfree) * bs})
		}
	}
	if out, err := exec.Command("netstat", "-ibn").Output(); err == nil {
		seen := map[string]bool{}
		for _, l := range strings.Split(string(out), "\n")[1:] {
			fs := strings.Fields(l)
			if len(fs) < 10 || seen[fs[0]] || fs[0] == "lo0" {
				continue
			}
			// rows with a link-level address carry the interface totals
			if !strings.HasPrefix(fs[2], "<Link#") {
				continue
			}
			seen[fs[0]] = true
			n := len(fs)
			c := netCounters{rxPackets: f(fs[n-7]), rxErr: f(fs[n-6]), rxBytes: f(fs[n-5]), txPackets: f(fs[n-4]), txErr: f(fs[n-3]), txBytes: f(fs[n-2])}
			r.nets[fs[0]] = c
			st.Nets = append(st.Nets, Net{Name: fs[0], Up: true, RxErrors: uint64(c.rxErr), TxErrors: uint64(c.txErr)})
		}
	}
	sort.Slice(st.Nets, func(i, j int) bool { return st.Nets[i].Name < st.Nets[j].Name })
	return r
}

func f(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }

func mb(s string) float64 {
	switch {
	case strings.HasSuffix(s, "M"):
		return f(strings.TrimSuffix(s, "M")) * (1 << 20)
	case strings.HasSuffix(s, "G"):
		return f(strings.TrimSuffix(s, "G")) * (1 << 30)
	}
	return f(s)
}

func uptimeSince(boot float64) float64 {
	var tv syscall.Timeval
	if syscall.Gettimeofday(&tv) != nil {
		return 0
	}
	return float64(tv.Sec) - boot
}

func sortDisks(*Stats) {}
