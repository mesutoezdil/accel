// Package procinfo enriches a PID with what `/proc` knows: owner, command
// line, and the container and pod it runs in (from the cgroup path).
package procinfo

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Info is what siltide could find out about a process.
type Info struct {
	User      string
	Command   string
	Container string // container ID, 64 hex characters when known
	PodUID    string // Kubernetes pod UID from the cgroup path
	Cgroup    string // raw `/proc/PID/cgroup` for other resolvers (Slurm)
	Started   time.Time
	App       map[string]float64 // metrics the process published about itself
}

var (
	rePodUID    = regexp.MustCompile(`pod([0-9a-f]{8}[_-][0-9a-f]{4}[_-][0-9a-f]{4}[_-][0-9a-f]{4}[_-][0-9a-f]{12})`)
	reContainer = regexp.MustCompile(`([0-9a-f]{64})`)
)

// Cache remembers lookups for a while; process metadata rarely changes.
type Cache struct {
	mu   sync.Mutex
	root string
	ttl  time.Duration
	seen map[int]entry
}

type entry struct {
	info Info
	at   time.Time
}

// New returns a cache reading from `/proc`.
func New(ttl time.Duration) *Cache { return &Cache{root: "/proc", ttl: ttl, seen: map[int]entry{}} }

// Lookup returns what is known about pid; fields stay empty when `/proc`
// does not say (other operating systems, vanished processes).
func (c *Cache) Lookup(pid int) Info {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.seen[pid]; ok && time.Since(e.at) < c.ttl {
		return e.info
	}
	info := read(c.root, pid)
	c.seen[pid] = entry{info, time.Now()}
	if len(c.seen) > 4096 {
		for k, e := range c.seen {
			if time.Since(e.at) >= c.ttl {
				delete(c.seen, k)
			}
		}
	}
	return info
}

// readOS replaces the `/proc` reader on systems without one.
var readOS func(pid int) Info

func read(root string, pid int) Info {
	if readOS != nil && root == "/proc" {
		return readOS(pid)
	}
	dir := root + "/" + strconv.Itoa(pid)
	var info Info
	if b, err := os.ReadFile(dir + "/cmdline"); err == nil {
		info.Command = strings.TrimSpace(strings.ReplaceAll(string(b), "\x00", " "))
	}
	if b, err := os.ReadFile(dir + "/status"); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if uid, ok := strings.CutPrefix(l, "Uid:"); ok {
				ids := strings.Fields(uid)
				if len(ids) == 0 {
					break
				}
				id := ids[0]
				info.User = id
				if u, err := user.LookupId(id); err == nil {
					info.User = u.Username
				}
				break
			}
		}
	}
	if b, err := os.ReadFile(dir + "/cgroup"); err == nil {
		info.Cgroup = string(b)
		info.Container, info.PodUID = ParseCgroup(info.Cgroup)
	}
	if b, err := os.ReadFile(dir + "/stat"); err == nil {
		info.Started = startTime(string(b), root)
	}
	info.App = appMetrics(pid)
	return info
}

// startTime turns field 22 of `/proc/PID/stat` (clock ticks since boot) into
// a wall-clock time using `/proc/uptime`.
func startTime(stat, root string) time.Time {
	i := strings.LastIndex(stat, ")")
	if i < 0 {
		return time.Time{}
	}
	f := strings.Fields(stat[i+1:])
	if len(f) < 20 {
		return time.Time{}
	}
	ticks, err := strconv.ParseFloat(f[19], 64)
	if err != nil {
		return time.Time{}
	}
	up, err := os.ReadFile(root + "/uptime")
	if err != nil {
		return time.Time{}
	}
	uptime, _ := strconv.ParseFloat(strings.Fields(string(up))[0], 64)
	const hz = 100 // USER_HZ, fixed at 100 in the Linux kernel ABI
	return time.Now().Add(-time.Duration((uptime - ticks/hz) * float64(time.Second)))
}

// appMetrics reads metrics a workload published about itself: a JSON
// object of numbers at `/run/siltide/app/<pid>.json` or
// `$TMPDIR/siltide-app-<pid>.json`, for example {"samples_per_s": 1830,
// "nccl_gbps": 210}. Nothing in the vendor stack reports these.
func appMetrics(pid int) map[string]float64 {
	for _, p := range []string{"/run/siltide/app/" + strconv.Itoa(pid) + ".json", filepath.Join(os.TempDir(), "siltide-app-"+strconv.Itoa(pid)+".json")} {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var m map[string]float64
		if json.Unmarshal(b, &m) == nil && len(m) > 0 {
			return m
		}
	}
	return nil
}

// ParseCgroup pulls the container ID and pod UID out of `/proc/PID/cgroup`.
// systemd writes the UID with underscores; Kubernetes uses dashes.
func ParseCgroup(text string) (container, pod string) {
	for _, l := range strings.Split(text, "\n") {
		path := l[strings.LastIndex(l, ":")+1:]
		if m := rePodUID.FindStringSubmatch(path); m != nil && pod == "" {
			pod = strings.ReplaceAll(m[1], "_", "-")
		}
		if m := reContainer.FindStringSubmatch(path); m != nil && container == "" {
			container = m[1]
		}
	}
	return container, pod
}
