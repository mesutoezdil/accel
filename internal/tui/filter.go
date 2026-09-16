package tui

import (
	"fmt"
	"strings"

	"github.com/mesutoezdil/accel/internal/device"
	"github.com/mesutoezdil/accel/internal/events"
)

// Filter is the search language: free words match names, pods, and users;
// key:value pairs narrow one field. Keys: dev (index or label), vendor,
// node, name, user, ns, pod, workload, container, sev, kind, and state.
type Filter struct {
	words []string
	keyed map[string][]string
}

// ParseFilter reads "python ns:ml dev:3".
func ParseFilter(s string) Filter {
	f := Filter{keyed: map[string][]string{}}
	for _, tok := range strings.Fields(strings.ToLower(s)) {
		if k, v, ok := strings.Cut(tok, ":"); ok && v != "" {
			switch k {
			case "gpu", "device", "idx":
				k = "dev"
			case "namespace":
				k = "ns"
			case "severity":
				k = "sev"
			}
			f.keyed[k] = append(f.keyed[k], v)
			continue
		}
		f.words = append(f.words, tok)
	}
	return f
}

// Empty reports whether the filter matches everything.
func (f Filter) Empty() bool { return len(f.words) == 0 && len(f.keyed) == 0 }

func (f Filter) want(key string, values ...string) bool {
	wants, ok := f.keyed[key]
	if !ok {
		return true
	}
	for _, w := range wants {
		for _, v := range values {
			if strings.Contains(strings.ToLower(v), w) {
				return true
			}
		}
	}
	return false
}

func (f Filter) words0(fields ...string) bool {
	for _, w := range f.words {
		hit := false
		for _, v := range fields {
			if strings.Contains(strings.ToLower(v), w) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

// Device reports whether d passes. Process-level keys (user, ns, pod, workload, and container)
// pass when any process of the device matches.
func (f Filter) Device(d device.Device) bool {
	if !f.want("dev", fmt.Sprint(d.Index), d.Label()) || !f.want("vendor", string(d.Vendor)) ||
		!f.want("node", d.Node, "local") || !f.want("name", d.Name) || !f.want("state", string(d.State)) {
		return false
	}
	for _, k := range []string{"user", "ns", "pod", "workload", "container"} {
		if _, ok := f.keyed[k]; ok {
			hit := false
			for _, p := range d.Procs {
				if f.Process(d, p) {
					hit = true
					break
				}
			}
			if !hit {
				return false
			}
		}
	}
	if len(f.words) == 0 {
		return true
	}
	fields := []string{d.Name, d.Node, string(d.Vendor), d.ID, d.Allocated}
	for _, p := range d.Procs {
		fields = append(fields, p.Name, p.User, p.Pod, p.Namespace, p.Workload)
	}
	return f.words0(fields...)
}

// Process reports whether p on d passes.
func (f Filter) Process(d device.Device, p device.Process) bool {
	return f.want("dev", fmt.Sprint(d.Index), d.Label()) && f.want("vendor", string(d.Vendor)) && f.want("node", d.Node, "local") &&
		f.want("user", p.User) && f.want("ns", p.Namespace) && f.want("pod", p.Pod) && f.want("workload", p.Workload) &&
		f.want("container", p.Container) && f.want("name", p.Name, d.Name) &&
		f.words0(p.Name, p.User, p.Command, p.Pod, p.Namespace, p.Workload, p.Container, fmt.Sprint(p.PID), d.Name)
}

// Event reports whether e passes.
func (f Filter) Event(e events.Event) bool {
	return f.want("sev", string(e.Severity)) && f.want("kind", e.Kind) && f.want("dev", e.Label, e.Device) &&
		f.words0(e.Message, e.Kind, string(e.Severity), e.Label)
}
