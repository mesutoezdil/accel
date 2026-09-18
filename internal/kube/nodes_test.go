package kube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeCluster serves the two endpoints the node view reads.
func fakeCluster(t *testing.T, nodes, pods any) *Resolver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/nodes":
			_ = json.NewEncoder(w).Encode(nodes)
		case r.URL.Path == "/api/v1/pods":
			_ = json.NewEncoder(w).Encode(pods)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r := &Resolver{pods: map[string]Pod{}, ttl: time.Minute, client: srv.Client(), apiURL: srv.URL, enabled: true, source: "kubeconfig"}
	return r
}

func TestNodesReadsWhatTheSchedulerBelieves(t *testing.T) {
	nodes := map[string]any{"items": []map[string]any{
		{
			"metadata": map[string]any{"name": "gpu-1", "labels": map[string]string{"nvidia.com/gpu.driver-version": "550.54.15"}},
			"spec":     map[string]any{"unschedulable": false},
			"status": map[string]any{
				"capacity":    map[string]string{"cpu": "128", "memory": "1Ti", "nvidia.com/gpu": "8", "hugepages-2Mi": "0"},
				"allocatable": map[string]string{"cpu": "127", "nvidia.com/gpu": "8"},
				"conditions":  []map[string]string{{"type": "Ready", "status": "True"}},
				"nodeInfo":    map[string]string{"containerRuntimeVersion": "containerd://1.7.13"},
			},
		},
		{
			"metadata": map[string]any{"name": "gpu-2"},
			"spec":     map[string]any{"unschedulable": true},
			"status": map[string]any{
				"capacity":    map[string]string{"huawei.com/Ascend910": "8"},
				"allocatable": map[string]string{"huawei.com/Ascend910": "4"},
				"conditions":  []map[string]string{{"type": "Ready", "status": "False"}},
			},
		},
		{ // no accelerators: not this table's business
			"metadata": map[string]any{"name": "cpu-only"},
			"status":   map[string]any{"capacity": map[string]string{"cpu": "8"}, "allocatable": map[string]string{"cpu": "8"}},
		},
	}}
	pods := map[string]any{"items": []map[string]any{
		{
			"metadata": map[string]any{"name": "train-0", "namespace": "ml", "uid": "u1"},
			"spec": map[string]any{"nodeName": "gpu-1", "containers": []map[string]any{
				{"name": "main", "resources": map[string]any{"requests": map[string]string{"nvidia.com/gpu": "4", "cpu": "8"}}},
			}},
			"status": map[string]any{"phase": "Running"},
		},
		{
			"metadata": map[string]any{"name": "infer-0", "namespace": "serve", "uid": "u2"},
			"spec": map[string]any{"nodeName": "gpu-1", "containers": []map[string]any{
				{"name": "main", "resources": map[string]any{"requests": map[string]string{"nvidia.com/gpu": "2"}}},
			}},
			"status": map[string]any{"phase": "Running"},
		},
	}}

	r := fakeCluster(t, nodes, pods)
	got := r.Nodes(context.Background())
	if len(got) != 2 {
		t.Fatalf("returned %d nodes, want the two with accelerators: %+v", len(got), got)
	}

	one := got[0]
	if one.Name != "gpu-1" || !one.Ready || one.Unschedulable {
		t.Errorf("node %+v", one)
	}
	if one.Driver != "550.54.15" || one.Runtime != "containerd://1.7.13" {
		t.Errorf("driver %q runtime %q", one.Driver, one.Runtime)
	}
	gpu := one.Resources["nvidia.com/gpu"]
	if gpu.Capacity != 8 || gpu.Allocatable != 8 {
		t.Errorf("gpu counts %+v", gpu)
	}
	if gpu.Requested != 6 || gpu.Pods != 2 {
		t.Errorf("requests summed to %d over %d pods, want 6 over 2", gpu.Requested, gpu.Pods)
	}
	if gpu.Free() != 2 {
		t.Errorf("free %d, want 2", gpu.Free())
	}
	if _, ok := one.Resources["cpu"]; ok {
		t.Error("cpu is not an accelerator")
	}
	if _, ok := one.Resources["hugepages-2Mi"]; ok {
		t.Error("hugepages are not an accelerator")
	}

	two := got[1]
	if !two.Unschedulable || two.Ready {
		t.Errorf("a cordoned, unready node read as %+v", two)
	}
	if a := two.Resources["huawei.com/Ascend910"]; a.Capacity != 8 || a.Allocatable != 4 {
		t.Errorf("ascend counts %+v", a)
	}
}

func TestNodesNeedsTheAPI(t *testing.T) {
	// A resolver reading only /var/log/pods has no view of the cluster.
	r := &Resolver{pods: map[string]Pod{}, ttl: time.Minute, enabled: true, source: "log-dir"}
	if got := r.Nodes(context.Background()); got != nil {
		t.Errorf("without the API it returned %+v", got)
	}
}

func TestIsAccelerator(t *testing.T) {
	for _, name := range []string{
		"nvidia.com/gpu", "nvidia.com/mig-3g.40gb", "amd.com/gpu", "huawei.com/Ascend910",
		"aws.amazon.com/neuron", "cambricon.com/mlu", "enflame.com/gcu", "hygon.com/dcu",
		"baidu.com/kunlunxin_xpu", "intel.com/gpu", "habana.ai/gaudi_hpu", "google.com/tpu",
	} {
		if !IsAccelerator(name) {
			t.Errorf("%s should count as an accelerator", name)
		}
	}
	for _, name := range []string{"cpu", "memory", "pods", "ephemeral-storage", "hugepages-1Gi", "example.com/widgets"} {
		if IsAccelerator(name) {
			t.Errorf("%s should not count as an accelerator", name)
		}
	}
}
