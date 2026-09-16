package replay

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReplay(t *testing.T) {
	p := filepath.Join(t.TempDir(), "rec.jsonl")
	rec := `{"time":"2026-09-16T10:00:00Z","host":"h","devices":[{"id":"nvidia-a","vendor":"nvidia","index":0,"name":"x","metrics":{"util":10}}]}
{"time":"2026-09-16T10:00:01Z","host":"h","devices":[{"id":"nvidia-a","vendor":"nvidia","index":0,"name":"x","metrics":{"util":20}}]}
`
	if err := os.WriteFile(p, []byte(rec), 0o644); err != nil {
		t.Fatal(err)
	}
	pr := Provider(p)
	if err := pr.Detect(); err != nil {
		t.Fatal(err)
	}
	devs, err := pr.Read(context.Background())
	if err != nil || len(devs) != 1 || devs[0].Metrics["util"] != 10 || devs[0].Node != "h" {
		t.Fatalf("%+v %v", devs, err)
	}
	if err := Provider(filepath.Join(t.TempDir(), "missing")).Detect(); err == nil {
		t.Fatal("missing file must fail")
	}
}
