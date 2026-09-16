package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(p, []byte("refresh: 2s\nthresholds:\n  temp_warn: 90\nnodes:\n  - name: a\n    url: http://a:9800\n  - name: b\n    ssh: root@b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p, true)
	if err != nil || c.Refresh != 2*time.Second || c.Thresholds.TempWarn != 90 || c.Thresholds.IdleUtil != 5 || len(c.Nodes) != 2 {
		t.Fatalf("%+v %v", c, err)
	}
	if _, err := Load(filepath.Join(dir, "missing.yaml"), true); err == nil {
		t.Fatal("explicit missing file must fail")
	}
	if _, err := Load(filepath.Join(dir, "missing.yaml"), false); err != nil {
		t.Fatal("default missing file is fine")
	}
	if err := os.WriteFile(p, []byte("refresh: 1s\nrefreshh: 2s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p, true); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("unknown key must name its line: %v", err)
	}
	if err := os.WriteFile(p, []byte("nodes:\n  - name: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p, true); err == nil {
		t.Fatal("node without url or ssh must fail")
	}
	if !Loopback("127.0.0.1:9800") || !Loopback("localhost:1") || Loopback(":9800") || Loopback("0.0.0.0:9800") {
		t.Fatal("loopback detection")
	}
	if !strings.Contains(Default().Print(), "refresh: 1s") {
		t.Fatal("print")
	}
}
