//go:build linux

package runner

import (
	"os/exec"
	"syscall"
)

// SetupProcessGroup mengonfigurasi process group terpisah dan Pdeathsig khusus Linux/Android
// sehingga jika parent Go terhenti, kernel otomatis mengirim SIGKILL ke child FFmpeg.
func SetupProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
}
