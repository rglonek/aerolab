//go:build windows

package file

import (
	"os"

	"golang.org/x/sys/windows"
)

// Windows locks byte ranges rather than whole files. Every caller locks the
// same one-byte range, so they still exclude each other.
const lockBytes = 1

func lockFile(f *os.File) error {
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, lockBytes, 0, new(windows.Overlapped))
}

func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, lockBytes, 0, new(windows.Overlapped))
}
