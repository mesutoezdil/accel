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

func TestLoadMergesADirectory(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// one concern per file, applied in name order
	write("conf/10-base.yaml", "refresh: 3s\ntheme: nord\n")
	write("conf/20-thresholds.yaml", "thresholds:\n  temp_warn: 70\n")
	write("conf/30-nodes.yaml", "nodes:\n  - name: a\n    url: http://a:9800\n")
	write("conf/40-theme.yaml", "theme: gruvbox\n")

	c, err := Load(filepath.Join(dir, "conf"), true)
	if err != nil {
		t.Fatal(err)
	}
	if c.Refresh != 3*time.Second || c.Theme != "gruvbox" || c.Thresholds.TempWarn != 70 || len(c.Nodes) != 1 {
		t.Fatalf("merged config %+v", c)
	}
	if c.Thresholds.IdleUtil != 5 {
		t.Errorf("a key no file sets should keep its default, got %v", c.Thresholds.IdleUtil)
	}

	if _, err := Load(filepath.Join(dir, "empty"), true); err == nil {
		t.Error("an explicit directory with no yaml in it must fail")
	}
	write("conf/50-bad.yaml", "refresh: 1s\nnope: 2\n")
	if _, err := Load(filepath.Join(dir, "conf"), true); err == nil || !strings.Contains(err.Error(), "50-bad.yaml") {
		t.Errorf("an unknown key must name the file it is in: %v", err)
	}
}

func TestLoadAppliesDropIns(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(main, []byte("refresh: 5s\ntheme: nord\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.d", "local.yaml"), []byte("theme: dracula\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := Files(main, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || filepath.Base(files[0]) != "config.yaml" || filepath.Base(files[1]) != "local.yaml" {
		t.Fatalf("files %v, want the main file then the drop-in", files)
	}
	c, err := Load(main, true)
	if err != nil {
		t.Fatal(err)
	}
	if c.Refresh != 5*time.Second || c.Theme != "dracula" {
		t.Fatalf("a drop-in should win over the main file: %+v", c)
	}

	// drop-ins alone, with no main file
	if err := os.Remove(main); err != nil {
		t.Fatal(err)
	}
	c, err = Load(main, false)
	if err != nil {
		t.Fatal(err)
	}
	if c.Theme != "dracula" {
		t.Fatalf("drop-ins should apply without a main file: %+v", c)
	}
}
