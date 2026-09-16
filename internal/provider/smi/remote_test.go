package smi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRemote drives the SSH-style runner with a fake shell: `command -v`
// probes answer with a path, and the tool invocation returns a fixture.
func TestRemote(t *testing.T) {
	efsmi, _ := os.ReadFile(filepath.Join("testdata", "efsmi.txt"))
	var seen []string
	run := func(_ context.Context, cmdline string) ([]byte, error) {
		seen = append(seen, cmdline)
		switch {
		case strings.HasPrefix(cmdline, "command -v efsmi"):
			return []byte("/usr/bin/efsmi\n"), nil
		case strings.HasPrefix(cmdline, "command -v "):
			return nil, os.ErrNotExist
		case strings.HasPrefix(cmdline, "/usr/bin/efsmi -q -d DEVICE,POWER,TEMP,MEMORY,USAGE,PCIE"):
			return efsmi, nil
		}
		return nil, os.ErrNotExist
	}
	var enflame *struct{ detect, read int }
	for _, p := range Remote("gpu07", run) {
		if p.Name != "node:gpu07:enflame" {
			if p.Detect() == nil {
				t.Errorf("%s detected on a host with only efsmi", p.Name)
			}
			continue
		}
		enflame = &struct{ detect, read int }{}
		if err := p.Detect(); err != nil {
			t.Fatal(err)
		}
		devs, err := p.Read(context.Background())
		if err != nil || len(devs) != 10 || devs[0].Node != "gpu07" || !strings.HasPrefix(devs[0].ID, "gpu07/enflame-") {
			t.Fatalf("%v %+v", err, devs[:1])
		}
	}
	if enflame == nil {
		t.Fatal("no enflame provider")
	}
	if shellQuote("a b") != "'a b'" || shellQuote("plain-1.0") != "plain-1.0" {
		t.Fatal("quoting")
	}
}
