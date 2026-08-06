//go:build unix

package file

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lockFile(f *os.File) error {
	return flock(f, unix.LOCK_EX)
}

func unlockFile(f *os.File) error {
	return flock(f, unix.LOCK_UN)
}

// flock retries on EINTR, which a blocking flock returns whenever the Go
// runtime delivers a signal to the waiting thread.
func flock(f *os.File, how int) error {
	for {
		err := unix.Flock(int(f.Fd()), how)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}
