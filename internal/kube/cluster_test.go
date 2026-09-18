package kube

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// These run against a real API server rather than a fixture, which is the
// only way to find out that a field moved, a path changed, or a response is
// shaped differently from what the test author imagined. They need a cluster
// and are skipped without one:
//
//	kind create cluster --name siltide
//	SILTIDE_TEST_CLUSTER=1 go test ./internal/kube/ -run Cluster -v
//
// CI creates a kind cluster and sets that variable. Nothing here needs an
// accelerator: the devices are siltide's own business, and what is being
// checked is the half that comes from Kubernetes.
func clusterOrSkip(t *testing.T) *Resolver {
	t.Helper()
	if os.Getenv("SILTIDE_TEST_CLUSTER") == "" {
		t.Skip("set SILTIDE_TEST_CLUSTER=1 with a cluster reachable to run this")
	}
	r := New(Options{Kubeconfig: os.Getenv("KUBECONFIG")})
	if !r.Enabled() {
		t.Fatalf("no pod source found; attempts: %+v", r.Attempts())
	}
	if r.Source() != "kubeconfig" && r.Source() != "in-cluster" {
		t.Skipf("source is %q, which reads no API server", r.Source())
	}
	return r
}

func TestClusterPodsAndNodes(t *testing.T) {
	r := clusterOrSkip(t)
	ctx := context.Background()

	pods := r.Pods()
	if len(pods) == 0 {
		t.Fatal("a running cluster has pods, even if only in kube-system")
	}
	var sample Pod
	for _, p := range pods {
		if p.Namespace == "kube-system" && len(p.Containers) > 0 {
			sample = p
			break
		}
	}
	if sample.Name == "" {
		t.Fatal("no kube-system pod with a container came back")
	}
	if sample.UID == "" || sample.Node == "" || sample.Phase == "" {
		t.Errorf("a pod is missing fields the interface shows: %+v", sample)
	}
	if got, ok := r.Lookup(sample.UID); !ok || got.Name != sample.Name {
		t.Errorf("looking a pod up by uid gave %+v %v", got, ok)
	}

	// The container id the pod carries is how a process on a device is
	// resolved to its container, so it has to be the bare id.
	for id := range sample.Containers {
		if strings.Contains(id, "/") || len(id) < 12 {
			t.Errorf("container id %q is not the bare id siltide matches against", id)
		}
	}

	// Nodes: a kind cluster advertises no accelerators, so the list is empty
	// and that is the correct answer, not a failure.
	nodes := r.Nodes(ctx)
	for _, n := range nodes {
		if len(n.Resources) == 0 {
			t.Errorf("node %s made the list with no accelerator resources", n.Name)
		}
	}
}

func TestClusterDescribeAndLogs(t *testing.T) {
	r := clusterOrSkip(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pod Pod
	for _, p := range r.Pods() {
		if p.Namespace == "kube-system" && p.Phase == "Running" && len(p.Containers) > 0 {
			pod = p
			break
		}
	}
	if pod.Name == "" {
		t.Skip("no running kube-system pod to describe")
	}

	text, err := r.Describe(ctx, pod.Namespace, pod.Name)
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	for _, want := range []string{pod.Name, pod.Namespace} {
		if !strings.Contains(text, want) {
			t.Errorf("describe does not mention %q:\n%s", want, first(text, 400))
		}
	}

	var container string
	for _, name := range pod.Containers {
		container = name
		break
	}
	logs, err := r.Logs(ctx, pod.Namespace, pod.Name, container, 5)
	if err != nil {
		// A container that has written nothing is not a failure of ours.
		if !strings.Contains(err.Error(), "HTTP 400") {
			t.Fatalf("logs: %v", err)
		}
	}
	if len(logs) > 0 && strings.Contains(logs, "HTTP") {
		t.Errorf("logs look like an error page:\n%s", first(logs, 200))
	}
}

// TestClusterKubectlAgrees compares what siltide read with what kubectl
// reports, so a difference shows up as a difference rather than as a number
// nobody checked.
func TestClusterKubectlAgrees(t *testing.T) {
	r := clusterOrSkip(t)
	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Skip("no kubectl to compare against")
	}
	out, err := exec.Command("kubectl", "get", "pods", "-A", "--no-headers").Output()
	if err != nil {
		t.Fatalf("kubectl: %v", err)
	}
	kubectl := len(strings.Split(strings.TrimSpace(string(out)), "\n"))
	ours := len(r.Pods())
	// Pods come and go between the two reads, so this is a sanity check on
	// the order of magnitude, not an equality.
	if ours == 0 || ours*2 < kubectl || kubectl*2 < ours {
		t.Errorf("siltide sees %d pods, kubectl %d", ours, kubectl)
	}
}

func first(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
