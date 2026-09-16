package smi

import (
	"os/exec"
	"syscall"
)

// killWithParent stops a background tool when accel dies without Close.
func killWithParent(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}
