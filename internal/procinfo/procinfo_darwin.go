package procinfo

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func init() { readOS = readDarwin }

// readDarwin asks `ps` for the user, start time, and command line; macOS
// has no `/proc`. One exec per process, cached by the caller for the TTL.
func readDarwin(pid int) Info {
	var info Info
	out, err := exec.Command("ps", "-o", "user=,lstart=,command=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return info
	}
	f := strings.Fields(string(out))
	if len(f) < 6 {
		return info
	}
	info.User = f[0]
	// lstart is 5 fields: "Wed Sep 17 08:40:12 2026"
	if t, err := time.ParseInLocation("Mon Jan 2 15:04:05 2006", strings.Join(f[1:6], " "), time.Local); err == nil {
		info.Started = t
	}
	info.Command = strings.Join(f[6:], " ")
	return info
}
