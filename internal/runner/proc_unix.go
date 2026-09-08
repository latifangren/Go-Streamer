//go:build !windows

package runner

import (
	"errors"
	"os/exec"
	"syscall"
	"time"
)

// KillProcessGroup mengirimkan sinyal SIGINT secara graceful ke seluruh Process Group child process.
func KillProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fallback ke direct process kill jika getpgid gagal
		return cmd.Process.Signal(syscall.SIGINT)
	}

	// Sinyal negatif (-pgid) dikirimkan ke seluruh grup proses
	return syscall.Kill(-pgid, syscall.SIGINT)
}

// ForceKillProcessGroup mengirimkan sinyal SIGKILL ke seluruh Process Group jika graceful kill timeout.
func ForceKillProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return cmd.Process.Kill()
	}

	return syscall.Kill(-pgid, syscall.SIGKILL)
}

// isProcessRunning memeriksa apakah process ID masih aktif di kernel
func isProcessRunning(pid int) bool {
	process, err := syscall.Getpgid(pid)
	if err != nil {
		return false
	}
	return process > 0
}

// WaitForExitGracefully menunggu proses selesai setelah diberi sinyal dengan toleransi waktu (timeout).
func WaitForExitGracefully(done chan struct{}, timeout time.Duration) error {
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return errors.New("timeout waiting for process group to terminate gracefully")
	}
}
