package host

import (
	"encoding/json"
	"testing"
	"time"
)

// TestSampleLive reads the real machine: it must not panic and must fill the
// basics on Linux and macOS.
func TestSampleLive(t *testing.T) {
	s := New()
	s.Sample()
	time.Sleep(200 * time.Millisecond)
	st := s.Sample()
	b, _ := json.Marshal(st)
	if st.Hostname == "" || st.Mem.Total <= 0 || len(b) < 100 {
		t.Fatalf("live sample: %s", b)
	}
	t.Logf("cpu %.0f%% load %.2f mem %.0fG nets %d fs %d", st.CPU.Percent, st.CPU.Load1, st.Mem.Total/(1<<30), len(st.Nets), len(st.Filesystems))
}
