//go:build !windows

package collect

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// cpuTime is the process's user plus system CPU time.
func cpuTime() (time.Duration, bool) {
	var ru syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &ru) != nil {
		return 0, false
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano()), true
}

// rssBytes reads the resident set from `/proc` on Linux; other systems fall
// back to the Go runtime's view.
func rssBytes() (float64, bool) {
	b, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0, false
	}
	f := strings.Fields(string(b))
	if len(f) < 2 {
		return 0, false
	}
	pages, err := strconv.ParseFloat(f[1], 64)
	if err != nil {
		return 0, false
	}
	return pages * float64(os.Getpagesize()), true
}
