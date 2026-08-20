package ingest

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func writeTar(t *testing.T, entries map[string][]byte, extra func(*tar.Writer)) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, body := range entries {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(body)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if extra != nil {
		extra(tw)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestUntarRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "out")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "evil")

	payload := writeTar(t, map[string][]byte{
		"../evil": []byte("pwned"),
	}, nil)
	err := untar(dest, bytes.NewReader(payload))
	if err == nil {
		t.Fatal("expected traversal error")
	}
	if _, err := os.Stat(outside); err == nil {
		t.Fatal("traversal wrote a file outside dest")
	}
}

func TestUntarRejectsAbsolutePath(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "out")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := writeTar(t, map[string][]byte{
		"/etc/passwd": []byte("nope"),
	}, nil)
	if err := untar(dest, bytes.NewReader(payload)); err == nil {
		t.Fatal("expected error for absolute archive path")
	}
}

func TestUntarSkipsSymlinkAndExtractsRegular(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "out")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := writeTar(t, map[string][]byte{
		"ok.txt": []byte("hello"),
	}, func(tw *tar.Writer) {
		hdr := &tar.Header{
			Name:     "link",
			Mode:     0o777,
			Typeflag: tar.TypeSymlink,
			Linkname: "/etc/passwd",
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
	})
	if err := untar(dest, bytes.NewReader(payload)); err != nil {
		t.Fatalf("untar: %s", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "ok.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}
	if fi, err := os.Lstat(filepath.Join(dest, "link")); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("symlink was planted in dest")
	}
}

func TestUntarNestedFile(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "out")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := writeTar(t, map[string][]byte{
		"tmp/collect_info_x.tgz": []byte("cf"),
	}, nil)
	if err := untar(dest, bytes.NewReader(payload)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "tmp", "collect_info_x.tgz"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "cf" {
		t.Fatalf("got %q", got)
	}
}

func TestUnzipRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "out")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(root, "evil.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("../evil")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "pwned"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := unzip(zipPath, dest); err == nil {
		t.Fatal("expected traversal error")
	}
	if _, err := os.Stat(filepath.Join(root, "evil")); err == nil {
		t.Fatal("traversal wrote a file outside dest")
	}
}

func TestUnzipContainedFile(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "out")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(root, "ok.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("sub/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	names, err := unzip(zipPath, dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 {
		t.Fatalf("names=%v", names)
	}
	got, err := os.ReadFile(filepath.Join(dest, "sub", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}
}
