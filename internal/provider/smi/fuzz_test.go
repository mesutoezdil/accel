package smi

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzParsers runs every text parser over whatever the fuzzer produces. The
// table-driven test beside this one holds them against the captured fixtures
// and a fixed set of mangled variants; this keeps looking, with coverage
// guiding it, for the input nobody thought to write down.
//
// The seeds are the fixtures themselves, so the fuzzer starts from text that
// already reaches deep into each parser instead of from noise.
func FuzzParsers(f *testing.F) {
	files, _ := filepath.Glob("testdata/*")
	for _, name := range files {
		// SOURCES says where the fixtures came from, and testdata/fuzz is the
		// corpus the fuzzer keeps for itself.
		if base := filepath.Base(name); base == "SOURCES" || base == "fuzz" {
			continue
		}
		b, err := os.ReadFile(name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Add([]byte(""))
	f.Add([]byte("|||"))

	specs := Specs()
	f.Fuzz(func(t *testing.T, in []byte) {
		for _, s := range specs {
			if s.Parse == nil {
				continue // sysfs readers have no text to parse
			}
			devs, err := s.Parse(in)
			if err != nil {
				continue
			}
			// Whatever comes back has to be a device siltide can render:
			// a metric map it can read and an index it can label.
			for _, d := range devs {
				if d.Metrics == nil {
					t.Fatalf("%s returned a device with no metric map from %q", s.Vendor, in)
				}
				if d.Index < 0 {
					t.Fatalf("%s returned device index %d from %q", s.Vendor, d.Index, in)
				}
				for _, p := range d.Procs {
					if p.Metrics == nil {
						t.Fatalf("%s returned a process with no metric map from %q", s.Vendor, in)
					}
				}
			}
		}
	})
}
