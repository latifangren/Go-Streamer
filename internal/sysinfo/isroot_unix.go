//go:build !windows

package sysinfo

import "os"

// IsRunningAsRoot memeriksa apakah proses berjalan dengan hak akses root/superuser (UID 0).
func IsRunningAsRoot() bool {
	return os.Geteuid() == 0
}
