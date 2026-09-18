package smi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParsersSurviveMangledOutput feeds every parser truncated and mangled
// versions of every captured fixture. A vendor tool that is interrupted, or
// prints a row with a column missing, must not take the collector down.
func TestParsersSurviveMangledOutput(t *testing.T) {
	files, _ := filepath.Glob("testdata/*")
	var inputs [][]byte
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil || strings.HasSuffix(f, "SOURCES") {
			continue
		}
		inputs = append(inputs, b)
		for _, cut := range []float64{0.9, 0.7, 0.5, 0.3, 0.1, 0.01} {
			inputs = append(inputs, b[:int(float64(len(b))*cut)])
		}
		// blank out every field of every line, one line at a time
		lines := strings.Split(string(b), "\n")
		for i := range lines {
			cp := append([]string(nil), lines...)
			cp[i] = strings.Repeat(" ", len(cp[i]))
			inputs = append(inputs, []byte(strings.Join(cp, "\n")))
			cp2 := append([]string(nil), lines...)
			cp2[i] = strings.Map(func(r rune) rune {
				if r >= '0' && r <= '9' {
					return ' '
				}
				return r
			}, cp2[i])
			inputs = append(inputs, []byte(strings.Join(cp2, "\n")))
		}
	}
	inputs = append(inputs, []byte(""), []byte("\n"), []byte("|||"), []byte("{"), []byte("<a>"))
	t.Logf("%d inputs", len(inputs))

	for _, s := range Specs() {
		if s.Parse == nil {
			continue // sysfs readers have no text to parse
		}
		for i, in := range inputs {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("%s panicked on input %d: %v\n%.200s", s.Vendor, i, r, in)
					}
				}()
				_, _ = s.Parse(in)
			}()
		}
	}
}
