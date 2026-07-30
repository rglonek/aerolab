//go:build integration_docker

// Package installertest holds helpers shared by the installer integration
// tests. Each installer suite under tests/installers is its own package, so
// shared helpers have to live in an importable package rather than a _test.go
// file.
package installertest

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// RequireDocker skips the test unless a reachable Docker daemon is available.
// The installer tests all run their generated script inside a container, so
// without Docker they can only be skipped, not run.
func RequireDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker binary not found in PATH; skipping installer test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "docker", "info").CombinedOutput(); err != nil {
		t.Skipf("docker daemon not reachable; skipping installer test: %v\n%s", err, string(out))
	}
}
