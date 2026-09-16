//go:build !linux

package smi

import "os/exec"

func killWithParent(*exec.Cmd) {}
