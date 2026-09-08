//go:build windows

package runner

import (
	"os/exec"
)

// SetupProcessGroup pada Windows merupakan operasi no-op karena Setpgid/Pdeathsig hanya didukung di Unix.
func SetupProcessGroup(cmd *exec.Cmd) {
	// No-op on Windows
}

// KillProcessGroup mematikan proses pada Windows.
func KillProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if err := cmd.Process.Kill(); err != nil && err.Error() != "os: process already finished" {
		return err
	}
	return nil
}

// ForceKillProcessGroup mematikan paksa proses pada Windows.
func ForceKillProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if err := cmd.Process.Kill(); err != nil && err.Error() != "os: process already finished" {
		return err
	}
	return nil
}
