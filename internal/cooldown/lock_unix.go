//go:build unix

package cooldown

import (
	"errors"
	"os"
	"syscall"
)

// tryLock takes an exclusive lock on f without blocking. The lock goes
// with the open file, so the OS releases it if the process dies.
func tryLock(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EINTR) {
		return false, nil
	}
	return err == nil, err
}

func unlock(f *os.File) {
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
