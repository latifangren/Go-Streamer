//go:build !linux && !windows

package runner

import (
	"os/exec"
	"syscall"
)

// SetupProcessGroup mengonfigurasi process group terpisah untuk platform POSIX non-Linux (seperti macOS/Darwin dan BSD)
// di mana Pdeathsig tidak didukung oleh kernel.
func SetupProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}
