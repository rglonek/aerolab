//go:build !nowebui

package cmd

import "testing"

func TestIsLoopbackListenAddr(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:3333", true},
		{"127.0.0.1", true},
		{"127.5.6.7:3333", true},
		{"[::1]:3333", true},
		{"::1", true},
		{"localhost:3333", true},
		{"0.0.0.0:3333", false},
		{"[::]:3333", false},
		{":3333", false},
		{"", false},
		{"*:3333", false},
		{"192.168.1.10:3333", false},
		{"10.0.0.1", false},
		{"example.invalid:3333", false},
	}
	for _, tt := range tests {
		if got := isLoopbackListenAddr(tt.addr); got != tt.want {
			t.Errorf("isLoopbackListenAddr(%q) = %v, want %v", tt.addr, got, tt.want)
		}
	}
}

func TestSafeJoinRemoteRejectsTraversal(t *testing.T) {
	const dest = "/opt/uploads"
	escapes := []string{
		"../evil",
		"../../etc/cron.d/evil",
		"sub/../../evil",
		"/etc/passwd",
		"/",
		"..",
		"..\\..\\evil",
		"",
		"   ",
	}
	for _, name := range escapes {
		if got, err := safeJoinRemote(dest, name); err == nil {
			t.Errorf("safeJoinRemote(%q, %q) = %q, expected an error", dest, name, got)
		}
	}
}

func TestSafeJoinRemoteAllowsContainedPaths(t *testing.T) {
	tests := []struct {
		dest  string
		entry string
		want  string
	}{
		{"/opt/uploads", "file.txt", "/opt/uploads/file.txt"},
		{"/opt/uploads", "./file.txt", "/opt/uploads/file.txt"},
		{"/opt/uploads", "sub/dir/file.txt", "/opt/uploads/sub/dir/file.txt"},
		{"/opt/uploads/", "file.txt", "/opt/uploads/file.txt"},
		{"/opt/uploads", "sub/./nested/file.txt", "/opt/uploads/sub/nested/file.txt"},
		// A ".." fully consumed by a preceding element stays inside.
		{"/opt/uploads", "a/b/c.txt", "/opt/uploads/a/b/c.txt"},
	}
	for _, tt := range tests {
		got, err := safeJoinRemote(tt.dest, tt.entry)
		if err != nil {
			t.Errorf("safeJoinRemote(%q, %q) returned an unexpected error: %s", tt.dest, tt.entry, err)
			continue
		}
		if got != tt.want {
			t.Errorf("safeJoinRemote(%q, %q) = %q, want %q", tt.dest, tt.entry, got, tt.want)
		}
	}
}
