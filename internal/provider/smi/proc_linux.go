package smi

import (
	"os/exec"
	"syscall"
)

// killWithParent stops a background tool when siltide dies without Close.
func killWithParent(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}
