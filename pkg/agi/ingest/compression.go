package ingest

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/rglonek/sbs"
	"github.com/xi2/xz"

	"github.com/nwaples/rardecode/v2"
)

// safeJoinLocal resolves an archive entry name against destDir and
// guarantees the result stays inside it.
//
// Entry names are attacker-controlled (S3/SFTP ingest, nested
// collectinfo archives), so an entry such as "../../etc/cron.d/evil"
// would otherwise be an arbitrary file write and a path to RCE.
func safeJoinLocal(destDir, entryName string) (string, error) {
	name := strings.TrimSpace(entryName)
	if name == "" {
		return "", fmt.Errorf("archive entry has an empty name")
	}
	// Archives use forward slashes; normalise Windows-style separators
	// so they cannot smuggle a traversal past the checks below.
	name = strings.ReplaceAll(name, "\\", "/")
	if path.IsAbs(name) {
		return "", fmt.Errorf("archive entry %q uses an absolute path", entryName)
	}
	for _, elem := range strings.Split(name, "/") {
		if elem == ".." {
			return "", fmt.Errorf("archive entry %q escapes the destination directory", entryName)
		}
	}

	cleanDest := filepath.Clean(destDir)
	rel := filepath.FromSlash(path.Clean(name))
	if rel == "." || rel == string(os.PathSeparator) {
		return "", fmt.Errorf("archive entry %q resolves to the destination directory itself", entryName)
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("archive entry %q uses an absolute path", entryName)
	}
	target := filepath.Join(cleanDest, rel)
	sep := string(os.PathSeparator)
	if target != cleanDest && !strings.HasPrefix(target, cleanDest+sep) {
		return "", fmt.Errorf("archive entry %q escapes the destination directory", entryName)
	}
	return target, nil
}

func archiveModeUnsafe(mode os.FileMode) bool {
	return mode&(os.ModeSymlink|os.ModeDevice|os.ModeNamedPipe|os.ModeSocket) != 0
}

func refuseExistingSymlink(target string) error {
	fi, err := os.Lstat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s: refusing to overwrite a symlink", target)
	}
	return nil
}

func unbz2(sourceFile string, destFile string) error {
	fd, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer fd.Close()
	decompressed := bzip2.NewReader(fd)
	fdDest, err := os.OpenFile(destFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fdDest.Close()
	_, err = io.Copy(fdDest, decompressed)
	if err != nil {
		return err
	}
	return nil
}

func isTarGz(file string) bool {
	fd, err := os.Open(file)
	if err != nil {
		return false
	}
	defer fd.Close()
	fdgzip, err := gzip.NewReader(fd)
	if err != nil {
		return false
	}
	defer fdgzip.Close()
	buffer := make([]byte, 4096)
	rdCnt, err := fdgzip.Read(buffer)
	if err != nil {
		return false
	}
	contentType := mimetype.Detect(buffer[0:rdCnt])
	return contentType.Is("application/x-tar")
}

func isTarBz(file string) bool {
	fd, err := os.Open(file)
	if err != nil {
		return false
	}
	defer fd.Close()
	fdgzip := bzip2.NewReader(fd)
	buffer := make([]byte, 4096)
	rdCnt, err := fdgzip.Read(buffer)
	if err != nil {
		return false
	}
	contentType := mimetype.Detect(buffer[0:rdCnt])
	return contentType.Is("application/x-tar")
}

func ungz(sourceFile string, destFile string) error {
	fd, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("open source file: %s", err)
	}
	defer fd.Close()
	decompressed, err := gzip.NewReader(fd)
	if err != nil {
		return fmt.Errorf("open gzip reader: %s", err)
	}
	fdDest, err := os.OpenFile(destFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open destination file: %s", err)
	}
	defer fdDest.Close()
	_, err = io.Copy(fdDest, decompressed)
	if err != nil {
		return fmt.Errorf("unpack: %s", err)
	}
	return nil
}

func unxz(sourceFile string, destFile string) error {
	fd, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("open source file: %s", err)
	}
	defer fd.Close()
	decompressed, err := xz.NewReader(fd, 0)
	if err != nil {
		return fmt.Errorf("open xz reader: %s", err)
	}
	fdDest, err := os.OpenFile(destFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open destination file: %s", err)
	}
	defer fdDest.Close()
	_, err = io.Copy(fdDest, decompressed)
	if err != nil {
		return fmt.Errorf("unpack: %s", err)
	}
	return nil
}

func unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {
		if archiveModeUnsafe(f.Mode()) {
			continue
		}

		fpath, err := safeJoinLocal(dest, f.Name)
		if err != nil {
			return filenames, err
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(fpath, os.ModePerm); err != nil {
				return filenames, err
			}
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}
		if err = refuseExistingSymlink(fpath); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode().Perm())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

func un7z(src string, dst string) error {
	if _, err := os.Stat(dst); err != nil {
		return err
	}
	if _, err := os.Stat(src); err != nil {
		return err
	}
	out, err := exec.Command("7z", "x", "-aou", "-y", fmt.Sprintf("-o%s", dst), src).CombinedOutput()
	if err != nil {
		return fmt.Errorf("err:%s .. out:%s", err, sbs.ByteSliceToString(out))
	}
	return nil
}

func untar(dst string, r io.Reader) error {

	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// Links/devices are skipped before join so a symlink named
		// "../evil" cannot fail (or plant) the rest of a legitimate archive.
		switch header.Typeflag {
		case tar.TypeDir, tar.TypeReg:
		default:
			continue
		}

		target, err := safeJoinLocal(dst, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {

		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			prevDir, _ := filepath.Split(target)
			if _, err := os.Stat(prevDir); os.IsNotExist(err) {
				if err := os.MkdirAll(prevDir, 0755); err != nil {
					return err
				}
			}
			if err := refuseExistingSymlink(target); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode).Perm())
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}

func unrar(src string, dst string) error {

	tr, err := rardecode.OpenReader(src)
	if err != nil {
		return err
	}
	defer tr.Close()

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		if archiveModeUnsafe(header.Mode()) {
			continue
		}

		target, err := safeJoinLocal(dst, header.Name)
		if err != nil {
			return err
		}

		if header.IsDir {
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		} else {
			targetDir := filepath.Dir(target)
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return err
			}
			if err := refuseExistingSymlink(target); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}
