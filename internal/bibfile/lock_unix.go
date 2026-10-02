//go:build !windows

package bibfile

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func tryLock(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	return err == nil, err
}

func releaseLock(file *os.File, name string) {
	// Remove while still locked. Waiters verify the pathname after acquiring.
	if name != "" {
		os.Remove(name)
	}
	unix.Flock(int(file.Fd()), unix.LOCK_UN)
	file.Close()
}
