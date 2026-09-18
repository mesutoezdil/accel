//go:build linux

package host

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

var (
	procRoot = "/proc"
	sysRoot  = "/sys"
)

func read(st *Stats) raw {
	r := raw{disks: map[string]diskCounters{}, nets: map[string]netCounters{}, ib: map[string]ibCounters{}}
	readCPU(st, &r)
	readMem(st)
	readLoad(st, &r)
	readFreq(st)
	readFilesystems(st)
	readDisks(&r)
	readNets(st, &r)
	readIB(st, &r)
	return r
}

func lines(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func f(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }

func readCPU(st *Stats, r *raw) {
	for _, l := range lines(procRoot + "/stat") {
		fs := strings.Fields(l)
		if len(fs) < 8 || !strings.HasPrefix(fs[0], "cpu") {
			continue
		}
		var t cpuTimes
		vals := []*float64{&t.user, &t.nice, &t.system, &t.idle, &t.iowait, &t.irq, &t.softirq, &t.steal}
		for i, v := range vals {
			if i+1 < len(fs) {
				*v = f(fs[i+1])
			}
		}
		r.cpu = append(r.cpu, t)
	}
	if n := len(r.cpu) - 1; n > 0 {
		st.CPU.Cores = n
	}
}

func readMem(st *Stats) {
	kv := map[string]float64{}
	for _, l := range lines(procRoot + "/meminfo") {
		k, v, ok := strings.Cut(l, ":")
		if ok {
			kv[k] = f(field(v, 0)) * 1024
		}
	}
	if t, ok := kv["MemTotal"]; ok {
		st.Mem.Total = Float(t)
		st.Mem.Available = Float(kv["MemAvailable"])
		st.Mem.Used = Float(t - kv["MemAvailable"])
		st.Mem.SwapTotal = Float(kv["SwapTotal"])
		st.Mem.SwapUsed = Float(kv["SwapTotal"] - kv["SwapFree"])
	}
}

func readLoad(st *Stats, r *raw) {
	if l := lines(procRoot + "/loadavg"); len(l) == 1 {
		fs := strings.Fields(l[0])
		if len(fs) >= 3 {
			st.CPU.Load1, st.CPU.Load5, st.CPU.Load15 = Float(f(fs[0])), Float(f(fs[1])), Float(f(fs[2]))
		}
	}
	if l := lines(procRoot + "/uptime"); len(l) == 1 {
		r.uptime = f(field(l[0], 0))
	}
}

func readFreq(st *Stats) {
	sum, n := 0.0, 0
	for _, l := range lines(procRoot + "/cpuinfo") {
		if k, v, ok := strings.Cut(l, ":"); ok && strings.TrimSpace(k) == "cpu MHz" {
			sum += f(strings.TrimSpace(v))
			n++
		}
	}
	if n == 0 {
		files, _ := filepath.Glob(sysRoot + "/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq")
		for _, p := range files {
			if l := lines(p); len(l) == 1 {
				sum += f(l[0]) / 1000
				n++
			}
		}
	}
	if n > 0 {
		st.CPU.MHz = Float(sum / float64(n))
	}
}

// virtual filesystems that only add noise
var skipFS = map[string]bool{"proc": true, "sysfs": true, "devtmpfs": true, "tmpfs": true, "cgroup": true, "cgroup2": true, "overlay": true,
	"squashfs": true, "devpts": true, "mqueue": true, "hugetlbfs": true, "debugfs": true, "tracefs": true, "securityfs": true,
	"pstore": true, "bpf": true, "configfs": true, "fusectl": true, "binfmt_misc": true, "autofs": true, "efivarfs": true, "rpc_pipefs": true, "nsfs": true}

func readFilesystems(st *Stats) {
	seen := map[string]bool{}
	for _, l := range lines(procRoot + "/mounts") {
		fs := strings.Fields(l)
		if len(fs) < 3 || skipFS[fs[2]] || seen[fs[1]] || strings.HasPrefix(fs[1], "/snap/") {
			continue
		}
		var sf syscall.Statfs_t
		if syscall.Statfs(fs[1], &sf) != nil || sf.Blocks == 0 {
			continue
		}
		seen[fs[1]] = true
		bs := float64(sf.Bsize)
		st.Filesystems = append(st.Filesystems, Filesystem{Mount: fs[1], Type: fs[2], Total: float64(sf.Blocks) * bs, Used: float64(sf.Blocks-sf.Bfree) * bs})
	}
}

func readDisks(r *raw) {
	for _, l := range lines(procRoot + "/diskstats") {
		fs := strings.Fields(l)
		if len(fs) < 14 {
			continue
		}
		name := fs[2]
		// whole devices only: skip partitions (sda1, nvme0n1p1) and loop/ram
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || isPartition(name) {
			continue
		}
		r.disks[name] = diskCounters{rd: f(fs[3]), rdSectors: f(fs[5]), wr: f(fs[7]), wrSectors: f(fs[9]), ioMs: f(fs[12])}
	}
}

func isPartition(name string) bool {
	_, err := os.Stat(sysRoot + "/block/" + name)
	return err != nil // partitions live under their parent, not `/sys/block`
}

func readNets(st *Stats, r *raw) {
	ls := lines(procRoot + "/net/dev")
	if len(ls) < 2 {
		return
	}
	for _, l := range ls[2:] {
		name, rest, ok := strings.Cut(l, ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "lo" {
			continue
		}
		fs := strings.Fields(rest)
		if len(fs) < 16 {
			continue
		}
		r.nets[name] = netCounters{rxBytes: f(fs[0]), rxPackets: f(fs[1]), rxErr: f(fs[2]), rxDrop: f(fs[3]), txBytes: f(fs[8]), txPackets: f(fs[9]), txErr: f(fs[10]), txDrop: f(fs[11])}
		up := false
		if s := lines(sysRoot + "/class/net/" + name + "/operstate"); len(s) == 1 {
			up = s[0] == "up"
		}
		c := r.nets[name]
		st.Nets = append(st.Nets, Net{Name: name, Up: up, RxErrors: uint64(c.rxErr), TxErrors: uint64(c.txErr), RxDrops: uint64(c.rxDrop), TxDrops: uint64(c.txDrop)})
	}
	sort.Slice(st.Nets, func(i, j int) bool { return st.Nets[i].Name < st.Nets[j].Name })
}

func readIB(st *Stats, r *raw) {
	ports, _ := filepath.Glob(sysRoot + "/class/infiniband/*/ports/*")
	for _, p := range ports {
		dev := filepath.Base(filepath.Dir(filepath.Dir(p)))
		port, _ := strconv.Atoi(filepath.Base(p))
		one := func(name string) float64 {
			if l := lines(p + "/" + name); len(l) == 1 {
				return f(field(l[0], 0))
			}
			return 0
		}
		state := ""
		if l := lines(p + "/state"); len(l) == 1 {
			if _, s, ok := strings.Cut(l[0], ": "); ok {
				state = s
			}
		}
		rate := ""
		if l := lines(p + "/rate"); len(l) == 1 {
			rate = l[0]
		}
		c := ibCounters{rx: one("counters/port_rcv_data"), tx: one("counters/port_xmit_data"),
			errs: one("counters/symbol_error") + one("counters/link_downed") + one("counters/port_rcv_errors")}
		r.ib[dev+"/"+itoa(port)] = c
		st.IB = append(st.IB, IBPort{Device: dev, Port: port, State: state, Rate: rate, Errors: uint64(c.errs)})
	}
}

func sortDisks(st *Stats) {
	sort.Slice(st.Disks, func(i, j int) bool { return st.Disks[i].Name < st.Disks[j].Name })
}
