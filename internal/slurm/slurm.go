// Package slurm maps processes and devices to Slurm jobs on HPC
// (high-performance computing) nodes: the job ID comes from the process
// cgroup, the job's name, user, and allocated devices from `scontrol`.
package slurm

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Job is a running Slurm job on this node.
type Job struct {
	ID      string
	Name    string
	User    string
	Devices []int // local device indices from the GRES (generic resource) allocation
}

var (
	reCgroupJob = regexp.MustCompile(`/job_(\d+)`)
	reGresIdx   = regexp.MustCompile(`\(IDX:([0-9,-]+)\)`)
)

// JobFromCgroup extracts the job ID from `/proc/PID/cgroup` text.
func JobFromCgroup(text string) string {
	if m := reCgroupJob.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	return ""
}

// Resolver caches `scontrol` output.
type Resolver struct {
	mu      sync.Mutex
	jobs    map[string]Job
	at      time.Time
	ttl     time.Duration
	enabled bool
	node    string
	run     func(ctx context.Context, args ...string) ([]byte, error)
}

// New returns a resolver; inert when `scontrol` is not installed.
func New(node string) *Resolver {
	r := &Resolver{jobs: map[string]Job{}, ttl: 30 * time.Second, node: node}
	if _, err := exec.LookPath("scontrol"); err == nil {
		r.enabled = true
		r.run = func(ctx context.Context, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, "scontrol", args...).Output()
		}
	}
	return r
}

// Enabled reports whether Slurm tools exist here.
func (r *Resolver) Enabled() bool { return r.enabled }

// Lookup returns the job with the given ID.
func (r *Resolver) Lookup(id string) (Job, bool) {
	if id == "" || !r.enabled {
		return Job{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refresh()
	j, ok := r.jobs[id]
	return j, ok
}

// Jobs returns every running job on the node.
func (r *Resolver) Jobs() []Job {
	if !r.enabled {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refresh()
	out := make([]Job, 0, len(r.jobs))
	for _, j := range r.jobs {
		out = append(out, j)
	}
	return out
}

func (r *Resolver) refresh() {
	if time.Since(r.at) < r.ttl {
		return
	}
	r.at = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := r.run(ctx, "show", "job", "--details", "--oneliner")
	if err != nil {
		return
	}
	r.jobs = Parse(string(out), r.node)
}

// Parse reads `scontrol show job --details --oneliner` and keeps the
// running jobs that touch node.
func Parse(text, node string) map[string]Job {
	jobs := map[string]Job{}
	for _, line := range strings.Split(text, "\n") {
		kv := map[string]string{}
		for _, f := range strings.Fields(line) {
			if k, v, ok := strings.Cut(f, "="); ok {
				kv[k] = v
			}
		}
		if kv["JobId"] == "" || kv["JobState"] != "RUNNING" {
			continue
		}
		if node != "" && kv["NodeList"] != "" && !strings.Contains(kv["NodeList"], node) && kv["Nodes"] != node {
			continue
		}
		j := Job{ID: kv["JobId"], Name: kv["JobName"], User: strings.Split(kv["UserId"], "(")[0]}
		// `GRES=gpu:2(IDX:0-1)` or `GRES=gpu(IDX:1,3)`
		if m := reGresIdx.FindStringSubmatch(line); m != nil {
			j.Devices = expand(m[1])
		}
		jobs[j.ID] = j
	}
	return jobs
}

// expand turns "0-1,3" into [0 1 3].
func expand(s string) []int {
	var out []int
	for _, part := range strings.Split(s, ",") {
		lo, hi, ok := strings.Cut(part, "-")
		a, err := strconv.Atoi(lo)
		if err != nil {
			continue
		}
		b := a
		if ok {
			b, _ = strconv.Atoi(hi)
		}
		for i := a; i <= b; i++ {
			out = append(out, i)
		}
	}
	return out
}
