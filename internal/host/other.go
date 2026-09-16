//go:build !linux && !darwin

package host

func read(*Stats) raw {
	return raw{disks: map[string]diskCounters{}, nets: map[string]netCounters{}, ib: map[string]ibCounters{}}
}

func sortDisks(*Stats) {}
