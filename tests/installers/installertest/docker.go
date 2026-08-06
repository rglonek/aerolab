//go:build integration_docker

// Package installertest holds helpers shared by the installer integration
// tests. Each installer suite under tests/installers is its own package, so
// shared helpers have to live in an importable package rather than a _test.go
// file.
package installertest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lithammer/shortuuid"
)

// runTimeout bounds a single container run. Installer scripts download packages
// from the internet, so this has to be generous, but it must still be bounded so
// a wedged pull cannot consume the whole `go test` timeout.
const runTimeout = 20 * time.Minute

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

// Arch returns the Docker architecture name for the host. The installer suites
// deliberately test only the native architecture: an arm64 host exercises arm64
// packages and an amd64 host exercises amd64 packages. Running amd64 containers
// under emulation tests package sets that would never be deployed on that host.
func Arch() string {
	if runtime.GOARCH == "arm64" {
		return "arm64"
	}
	return "amd64"
}

// Platform returns the value to pass to `docker run --platform`. Passing it
// explicitly is required rather than cosmetic: without it Docker resolves a
// multi-arch tag against the host platform and exits 125 when the tag is a
// manifest list with no entry for that platform, instead of falling back the way
// it does for single-manifest images.
func Platform() string {
	return "linux/" + Arch()
}

// imageNamespace is the registry namespace holding per-architecture image
// mirrors. Docker Hub and quay.io/centos both use "amd64" and "arm64v8".
func imageNamespace() string {
	if Arch() == "arm64" {
		return "arm64v8"
	}
	return "amd64"
}

// Image expands the "{arch}" placeholder in an image reference to the
// per-architecture namespace for the host, so a single matrix entry such as
// "{arch}/rockylinux:9" or "quay.io/centos/{arch}:stream8" resolves to the
// native image on both amd64 and arm64.
func Image(ref string) string {
	return strings.ReplaceAll(ref, "{arch}", imageNamespace())
}

// RunScript writes script into a scratch directory bind-mounted at /mnt inside
// image, runs it with bash, and returns the combined output. The container is
// pinned to the host platform. The scratch directory is managed by the testing
// package, so nothing is left behind in the source tree even when the test
// fails.
func RunScript(t *testing.T, image string, script []byte) (string, error) {
	t.Helper()

	dir := t.TempDir()
	name := shortuuid.New()
	if err := os.WriteFile(filepath.Join(dir, name+".sh"), script, 0o755); err != nil {
		return "", fmt.Errorf("could not write install script: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", "run",
		"--platform", Platform(),
		"-v", dir+":/mnt",
		"--rm", "-i",
		"--name", name,
		image,
		"/bin/bash", "-c", fmt.Sprintf("chmod +x /mnt/%s.sh && /mnt/%s.sh", name, name),
	).CombinedOutput()
	return string(out), err
}

// RunScriptInImage resolves the "{arch}" placeholder in ref and runs the script
// in the resulting image, reporting the outcome through t. It returns false if
// the run failed, so a caller iterating over a matrix can record the failure and
// keep going instead of aborting on the first bad image.
func RunScriptInImage(t *testing.T, ref string, script []byte) bool {
	t.Helper()

	image := Image(ref)
	t.Logf("running installer script in %s (%s)", image, Platform())
	out, err := RunScript(t, image, script)
	if err != nil {
		t.Errorf("installer script failed in %s (%s): %v\n%s", image, Platform(), err, out)
		return false
	}
	t.Logf("installer script succeeded in %s\n%s", image, out)
	return true
}
