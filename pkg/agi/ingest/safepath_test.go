package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeJoinLocalRejectsTraversal(t *testing.T) {
	const dest = "/opt/ingest/out"
	escapes := []string{
		"../evil",
		"../../etc/cron.d/evil",
		"sub/../../evil",
		"/etc/passwd",
		"/",
		"..",
		`..\..\evil`,
		"",
		"   ",
		".",
	}
	for _, name := range escapes {
		if got, err := safeJoinLocal(dest, name); err == nil {
			t.Errorf("safeJoinLocal(%q, %q) = %q, expected an error", dest, name, got)
		}
	}
}

func TestSafeJoinLocalAllowsContainedPaths(t *testing.T) {
	tests := []struct {
		dest  string
		entry string
		want  string
	}{
		{"/opt/ingest/out", "file.txt", "/opt/ingest/out/file.txt"},
		{"/opt/ingest/out", "./file.txt", "/opt/ingest/out/file.txt"},
		{"/opt/ingest/out", "sub/dir/file.txt", "/opt/ingest/out/sub/dir/file.txt"},
		{"/opt/ingest/out/", "file.txt", "/opt/ingest/out/file.txt"},
		{"/opt/ingest/out", "sub/./nested/file.txt", "/opt/ingest/out/sub/nested/file.txt"},
		{"/opt/ingest/out", "tmp/collect_info_x.tgz", "/opt/ingest/out/tmp/collect_info_x.tgz"},
	}
	for _, tt := range tests {
		got, err := safeJoinLocal(tt.dest, tt.entry)
		if err != nil {
			t.Errorf("safeJoinLocal(%q, %q) returned an unexpected error: %s", tt.dest, tt.entry, err)
			continue
		}
		if got != tt.want {
			t.Errorf("safeJoinLocal(%q, %q) = %q, want %q", tt.dest, tt.entry, got, tt.want)
		}
	}
}

func TestRegularFileInsideRoot(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "cf.tgz")
	if err := os.WriteFile(inside, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := regularFileInsideRoot(root, inside); err != nil {
		t.Fatalf("contained regular file rejected: %s", err)
	}

	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := regularFileInsideRoot(root, outside); err == nil {
		t.Fatal("expected error for path outside root")
	}

	link := filepath.Join(root, "link.tgz")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := regularFileInsideRoot(root, link); err == nil {
		t.Fatal("expected error for symlink inside root")
	}

	viaLinkDir := t.TempDir()
	realDir := filepath.Join(viaLinkDir, "real")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outsideFile := filepath.Join(t.TempDir(), "passwd")
	if err := os.WriteFile(outsideFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(outsideFile), filepath.Join(realDir, "escape")); err != nil {
		t.Fatal(err)
	}
	escaped := filepath.Join(realDir, "escape", filepath.Base(outsideFile))
	if err := regularFileInsideRoot(realDir, escaped); err == nil {
		t.Fatal("expected error when a symlink dir points asadm at a file outside root")
	}
}

func TestDestInsideRoot(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "n1_cf.tgz")
	if err := destInsideRoot(root, dest); err != nil {
		t.Fatalf("contained dest rejected: %s", err)
	}
	if err := destInsideRoot(root, filepath.Join(root, "..", "evil.tgz")); err == nil {
		t.Fatal("expected error for dest that escapes root")
	}
}

func TestCollectInfoRenamePath(t *testing.T) {
	got, ok := collectInfoRenamePath("/data/collectinfo/x1_cf.tgz", "n1")
	if !ok {
		t.Fatal("expected ok")
	}
	if got != "/data/collectinfo/n1_x1_cf.tgz" {
		t.Fatalf("got %q", got)
	}
	if _, ok := collectInfoRenamePath("/data/collectinfo/x1_cf.tgz", ".."); ok {
		t.Fatal("expected reject for .. prefix")
	}
	if _, ok := collectInfoRenamePath("/data/collectinfo/x1_cf.tgz", "a/b"); ok {
		t.Fatal("expected reject for slash in prefix")
	}
	if _, ok := collectInfoRenamePath("/data/collectinfo/x1_cf.tgz", `a\b`); ok {
		t.Fatal("expected reject for backslash in prefix")
	}
}

func TestIsSafeNamePrefix(t *testing.T) {
	if !isSafeNamePrefix("n1") || !isSafeNamePrefix("BB9C2") {
		t.Fatal("expected safe prefixes")
	}
	for _, p := range []string{"", ".", "..", "../x", "a/b", `a\b`} {
		if isSafeNamePrefix(p) {
			t.Errorf("isSafeNamePrefix(%q) = true, want false", p)
		}
	}
}

func TestProcessCollectInfoFileAsadmRejectsOutsidePath(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.tgz")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	i := &Ingest{config: &Config{}}
	i.config.Directories.CollectInfo = root
	err := i.processCollectInfoFileAsadm(outside, &cfContents{}, nil)
	if err == nil {
		t.Fatal("expected asadm to refuse a path outside collectinfo")
	}

	link := filepath.Join(root, "link.tgz")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := i.processCollectInfoFileAsadm(link, &cfContents{}, nil); err == nil {
		t.Fatal("expected asadm to refuse a symlink")
	}
}
