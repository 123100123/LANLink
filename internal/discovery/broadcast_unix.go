//go:build !windows

package discovery

import "syscall"

// controlBroadcast enables SO_BROADCAST on the raw socket fd so the beacon may
// send to broadcast addresses.
func controlBroadcast(fd uintptr) error {
	return syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
}
