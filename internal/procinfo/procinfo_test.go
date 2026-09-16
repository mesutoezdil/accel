package procinfo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCgroup(t *testing.T) {
	cases := []struct{ in, container, pod string }{
		{"0::/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod0a1b2c3d_1111_2222_3333_444455556666.slice/cri-containerd-" + h(64, 'a') + ".scope\n",
			h(64, 'a'), "0a1b2c3d-1111-2222-3333-444455556666"},
		{"12:pids:/kubepods/besteffort/pod0a1b2c3d-1111-2222-3333-444455556666/" + h(64, 'b') + "\n", h(64, 'b'), "0a1b2c3d-1111-2222-3333-444455556666"},
		{"0::/user.slice/user-1000.slice/session-2.scope\n", "", ""},
		{"0::/docker/" + h(64, 'c') + "\n", h(64, 'c'), ""},
	}
	for _, c := range cases {
		container, pod := ParseCgroup(c.in)
		if container != c.container || pod != c.pod {
			t.Errorf("%q: got %q %q", c.in, container, pod)
		}
	}
}

func h(n int, c byte) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

func TestStartTimeAndApp(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "uptime"), []byte("1000.00 900.00\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stat := "42 (my prog) S 1 42 42 0 -1 4194560 100 0 0 0 5 3 0 0 20 0 1 0 90000 1000 200 18446744073709551615"
	st := startTime(stat, root)
	if age := time.Since(st).Seconds(); age < 99 || age > 101 { // 1000 s up, started at tick 90000 = 900 s
		t.Fatalf("age %v", age)
	}
	t.Setenv("TMPDIR", root)
	if err := os.WriteFile(filepath.Join(root, "accel-app-7.json"), []byte(`{"samples_per_s": 1830}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if m := appMetrics(7); m["samples_per_s"] != 1830 {
		t.Fatalf("app %v", m)
	}
}
