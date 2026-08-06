package file

import (
	"fmt"
	"os"
	"path/filepath"
)

// Lock is an exclusive advisory lock on a file, held until Release is called.
//
// It excludes every other holder of the same path, in this process and in any
// other. It does not nest: taking the same path twice without releasing it
// deadlocks, so where goroutines race as well, serialise them on a mutex and
// take the file lock once behind it.
type Lock struct {
	f *os.File
}

// AcquireLock creates path if it is not there and blocks until the lock is
// held.
func AcquireLock(path string) (*Lock, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create lock directory %s: %w", dir, err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file %s: %w", path, err)
	}
	if err := lockFile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to lock %s: %w", path, err)
	}
	return &Lock{f: f}, nil
}

// Release unlocks and closes the lock file. The file is left on disk on
// purpose: deleting it would let the next process lock a path that the holder
// of the old one no longer shares.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	f := l.f
	l.f = nil
	err := unlockFile(f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
