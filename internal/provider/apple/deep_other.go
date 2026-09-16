//go:build !(darwin && arm64)

package apple

import "errors"

func loadDeep() error                 { return errors.New("not an Apple silicon Mac") }
func power() (float64, float64, bool) { return 0, 0, false }
func temperature() (float64, bool)    { return 0, false }
func gpuTime() map[int]procTime       { return nil }

type procTime struct {
	name string
	ns   float64
}
