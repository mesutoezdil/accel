//go:build windows

package collect

import "time"

func cpuTime() (time.Duration, bool) { return 0, false }
func rssBytes() (float64, bool)      { return 0, false }
