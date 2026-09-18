package kube

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
)

// NodeInfo is what the cluster believes about one node's accelerators, which
// is a different question from what the devices on it report. The scheduler
// hands out whole devices by count; siltide watches what those devices then
// do. The two disagreeing is the interesting case.
type NodeInfo struct {
	Name          string                  `json:"name"`
	Ready         bool                    `json:"ready"`
	Unschedulable bool                    `json:"unschedulable,omitempty"`
	Resources     map[string]NodeResource `json:"resources"` // "nvidia.com/gpu" and the like
	Driver        string                  `json:"driver,omitempty"`
	Runtime       string                  `json:"runtime,omitempty"`
}

// NodeResource is one accelerator resource on one node.
type NodeResource struct {
	Capacity    int64 `json:"capacity"`    // what the node says it has
	Allocatable int64 `json:"allocatable"` // what the scheduler may hand out
	Requested   int64 `json:"requested"`   // what pods on it have asked for
	Pods        int   `json:"pods"`        // how many pods those requests come from
}

// Free is what the scheduler would still place here.
func (r NodeResource) Free() int64 { return max64(r.Allocatable-r.Requested, 0) }

// standardResources are the ones every node has and nobody means when they
// say accelerator.
var standardResources = map[string]bool{
	"cpu": true, "memory": true, "pods": true, "ephemeral-storage": true,
	"storage": true, "attachable-volumes-aws-ebs": true,
}

// acceleratorWords are what a vendor calls its device in a resource name.
var acceleratorWords = []string{"gpu", "npu", "tpu", "accelerator", "neuron", "mlu", "gcu", "dcu", "xpu", "vpu", "ascend", "hpu"}

// acceleratorDomains are vendors whose resources are devices whatever they
// are called. nvidia.com/mig-3g.40gb is a slice of a GPU and says neither
// gpu nor npu anywhere in its name.
var acceleratorDomains = []string{
	"nvidia.com/", "amd.com/", "intel.com/", "huawei.com/", "aws.amazon.com/",
	"cambricon.com/", "enflame.com/", "hygon.com/", "baidu.com/", "habana.ai/",
	"metax-tech.com/", "mthreads.com/", "iluvatar.ai/", "vastaitech.com/", "biren.com/",
}

// IsAccelerator reports whether a Kubernetes resource name is a device
// siltide has any business reporting on.
func IsAccelerator(name string) bool {
	if standardResources[name] || strings.HasPrefix(name, "hugepages-") {
		return false
	}
	lower := strings.ToLower(name)
	for _, w := range acceleratorWords {
		if strings.Contains(lower, w) {
			return true
		}
	}
	for _, d := range acceleratorDomains {
		if strings.HasPrefix(lower, d) {
			return true
		}
	}
	return false
}

// nodeJSON is the part of a node siltide reads.
type nodeJSON struct {
	Metadata struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		Unschedulable bool `json:"unschedulable"`
	} `json:"spec"`
	Status struct {
		Capacity    map[string]string `json:"capacity"`
		Allocatable map[string]string `json:"allocatable"`
		Conditions  []struct {
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"conditions"`
		NodeInfo struct {
			ContainerRuntime string `json:"containerRuntimeVersion"`
			KernelVersion    string `json:"kernelVersion"`
		} `json:"nodeInfo"`
	} `json:"status"`
}

// Nodes returns what the API server says about accelerators on each node,
// with the pod requests already summed onto them. It needs the API: a
// resolver reading only /var/log/pods has no view of the cluster and returns
// nothing, which is the honest answer rather than an empty table.
func (r *Resolver) Nodes(ctx context.Context) []NodeInfo {
	if !r.enabled || r.apiURL == "" {
		return nil
	}
	r.mu.Lock()
	fresh := time.Since(r.nodesAt) < r.ttl && r.nodes != nil
	cached := r.nodes
	r.mu.Unlock()
	if fresh {
		return cached
	}

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var list struct {
		Items []nodeJSON `json:"items"`
	}
	if err := r.get(ctx, "/api/v1/nodes?limit=500", &list); err != nil {
		return nil
	}

	byName := map[string]*NodeInfo{}
	var order []string
	for _, it := range list.Items {
		n := &NodeInfo{
			Name:          it.Metadata.Name,
			Unschedulable: it.Spec.Unschedulable,
			Resources:     map[string]NodeResource{},
			Runtime:       it.Status.NodeInfo.ContainerRuntime,
			Driver:        driverLabel(it.Metadata.Labels),
		}
		for _, c := range it.Status.Conditions {
			if c.Type == "Ready" {
				n.Ready = c.Status == "True"
			}
		}
		for name, q := range it.Status.Capacity {
			if !IsAccelerator(name) {
				continue
			}
			res := n.Resources[name]
			res.Capacity = quantity(q)
			n.Resources[name] = res
		}
		for name, q := range it.Status.Allocatable {
			if !IsAccelerator(name) {
				continue
			}
			res := n.Resources[name]
			res.Allocatable = quantity(q)
			n.Resources[name] = res
		}
		byName[n.Name] = n
		order = append(order, n.Name)
	}

	// What the pods on each node have asked for, which is the number the
	// scheduler subtracts from allocatable.
	for _, p := range r.Pods() {
		n := byName[p.Node]
		if n == nil {
			continue
		}
		for name, q := range p.Requests {
			if !IsAccelerator(name) {
				continue
			}
			res := n.Resources[name]
			res.Requested += quantity(q)
			res.Pods++
			n.Resources[name] = res
		}
	}

	out := make([]NodeInfo, 0, len(order))
	for _, name := range order {
		n := byName[name]
		if len(n.Resources) == 0 {
			continue // a node with no accelerators is not this table's business
		}
		out = append(out, *n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	r.mu.Lock()
	r.nodes, r.nodesAt = out, time.Now()
	r.mu.Unlock()
	return out
}

// driverLabel picks the driver version a device plugin advertises, so the
// table can say which node is running something different from the others.
func driverLabel(labels map[string]string) string {
	for _, k := range []string{
		"nvidia.com/cuda.driver.major",
		"nvidia.com/gpu.driver-version",
		"nvidia.com/cuda.driver-version.full",
		"feature.node.kubernetes.io/nvidia-driver.version",
		"amd.com/gpu.driver-version",
	} {
		if v := labels[k]; v != "" {
			return v
		}
	}
	return ""
}

// quantity reads the whole-number resource quantities device plugins use.
// Accelerators are handed out whole, so anything with a fraction or a suffix
// is not a device count and reads as none.
func quantity(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
