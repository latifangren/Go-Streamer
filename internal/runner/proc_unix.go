//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

// SetupProcessGroup mengonfigurasi process group terpisah dan sinyal pematian otomatis saat parent mati (anti-zombie).
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

// KillProcessGroup mengirim sinyal SIGINT ke seluruh process group (-pgid) untuk graceful stop.
func KillProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fallback mengirim sinyal langsung ke proses jika pgid gagal diperoleh
		return cmd.Process.Signal(syscall.SIGINT)
	}
	if err := syscall.Kill(-pgid, syscall.SIGINT); err != nil {
		if err == syscall.ESRCH {
			return nil
		}
		return err
	}
	return nil
}

// ForceKillProcessGroup mengirim sinyal SIGKILL ke seluruh process group (-pgid).
func ForceKillProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fallback mematikan langsung proses jika pgid gagal diperoleh
		return cmd.Process.Kill()
	}
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		if err == syscall.ESRCH {
			return nil
		}
		return err
	}
	return nil
}
