package ingest

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func pathInsideRoot(root, p string) error {
	if strings.TrimSpace(p) == "" {
		return fmt.Errorf("empty path")
	}
	absRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(filepath.Clean(p))
	if err != nil {
		return err
	}
	sep := string(os.PathSeparator)
	if absPath != absRoot && !strings.HasPrefix(absPath, absRoot+sep) {
		return fmt.Errorf("path %q is outside %q", p, root)
	}
	return nil
}

func resolvedInsideRoot(root, p string) error {
	if err := pathInsideRoot(root, p); err != nil {
		return err
	}
	absRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(filepath.Clean(p))
	if err != nil {
		return err
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return fmt.Errorf("resolve root %q: %w", root, err)
	}
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return fmt.Errorf("resolve path %q: %w", p, err)
	}
	sep := string(os.PathSeparator)
	if resolvedPath != resolvedRoot && !strings.HasPrefix(resolvedPath, resolvedRoot+sep) {
		return fmt.Errorf("path %q escapes %q via symlink", p, root)
	}
	return nil
}

// regularFileInsideRoot reports whether p is a regular file (not a
// symlink) whose resolved path stays inside root. Collectinfo paths
// come from walking uploaded archives and from progress.json, so a
// ZipSlipped symlink or a planted progress entry would otherwise be
// handed to asadm as -cf.
func regularFileInsideRoot(root, p string) error {
	fi, err := os.Lstat(p)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("path %q is a symlink", p)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("path %q is not a regular file", p)
	}
	return resolvedInsideRoot(root, p)
}

// destInsideRoot checks a not-yet-created rename destination. The
// parent must already exist and stay inside root after symlink
// resolution so a prefix such as "../evil" cannot move a collectinfo
// file out of the ingest tree.
func destInsideRoot(root, dest string) error {
	if err := pathInsideRoot(root, dest); err != nil {
		return err
	}
	parent := filepath.Dir(dest)
	fi, err := os.Lstat(parent)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("parent of %q is a symlink", dest)
	}
	if !fi.IsDir() {
		return fmt.Errorf("parent of %q is not a directory", dest)
	}
	return resolvedInsideRoot(root, parent)
}

func isSafeNamePrefix(prefix string) bool {
	if prefix == "" || prefix == "." || prefix == ".." {
		return false
	}
	if strings.ContainsAny(prefix, `/\`) {
		return false
	}
	return filepath.Base(prefix) == prefix
}

func collectInfoRenamePath(filePath, prefix string) (string, bool) {
	if !isSafeNamePrefix(prefix) {
		return "", false
	}
	_, ffile := path.Split(filePath)
	if ffile == "" || ffile == "." || ffile == ".." || !isSafeNamePrefix(ffile) {
		return "", false
	}
	return path.Join(path.Dir(filePath), prefix+"_"+ffile), true
}
