//go:build integration_docker || integration_cloud

// Package e2e_test contains end-to-end integration tests that drive the real
// aerolab binary. They replace the legacy tests/cli/test.sh suite.
//
// There are two tiers, selected by build tag:
//
//   - integration_docker : Docker-backend tests, CI-runnable on any Linux/mac
//     host with Docker. Run with `make test-docker` or
//     `go test -tags=integration_docker ./tests/e2e/...`.
//   - integration_cloud  : AWS/GCP + Aerospike Cloud tests. These need real
//     cloud credentials and are opt-in/manual. Run with
//     `make test-cloud` or `go test -tags=integration_cloud ./tests/e2e/...`.
//
// Requirements:
//   - The aerolab binary. By default the harness builds it once from source into
//     a temp dir; set AEROLAB_BIN=/path/to/aerolab to use a prebuilt binary.
//   - Docker tier: a running Docker daemon (tests skip if `docker` is missing or
//     the daemon is unreachable).
//
// Optional environment overrides (no hardcoded machine-specific paths):
//   - AEROLAB_BIN                    prebuilt aerolab binary to use instead of building
//   - AEROLAB_FEATURES_FILE          path to an Aerospike features file (needed for
//                                    Enterprise images); wired via
//                                    `config defaults -k '*.FeaturesFilePath'`
//   - AEROLAB_E2E_DISTRO             base distro (default: ubuntu)
//   - AEROLAB_E2E_DISTRO_VER         distro version (default: 24.04)
//   - AEROLAB_E2E_ASVER              Aerospike version selector (default: 8.*)
//   - AEROLAB_E2E_OS_MATRIX          set to run the multi-distro OS matrix test
//   - AEROLAB_E2E_CLOUD              set to run the Aerospike Cloud tier
//   - AEROLAB_E2E_AWS_REGION         AWS region for the cloud tier (default: us-east-1)
//   - AEROLAB_E2E_VPC_ID             VPC id used when creating a cloud database
package e2e_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	buildOnce sync.Once
	builtBin  string
	buildErr  error
)

// aerolabBinary returns the path to an aerolab binary, building it once per test
// run if AEROLAB_BIN is not provided. On build failure it returns the error so
// callers can skip.
func aerolabBinary(t *testing.T) string {
	t.Helper()
	if b := os.Getenv("AEROLAB_BIN"); b != "" {
		return b
	}
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "aerolab-e2e-bin-")
		if err != nil {
			buildErr = err
			return
		}
		out := filepath.Join(dir, "aerolab")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		// Build the CLI main package. Inherit the caller's environment but force
		// vendored, workspace-off builds to match the Makefile.
		cmd := exec.CommandContext(ctx, "go", "build", "-o", out, "github.com/aerospike/aerolab/cli")
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=vendor")
		if b, err := cmd.CombinedOutput(); err != nil {
			buildErr = &buildFailure{output: string(b), err: err}
			return
		}
		builtBin = out
	})
	if buildErr != nil {
		t.Skipf("aerolab binary unavailable (set AEROLAB_BIN to skip building): %v", buildErr)
	}
	return builtBin
}

type buildFailure struct {
	output string
	err    error
}

func (b *buildFailure) Error() string { return b.err.Error() + "\n" + b.output }

// requireDocker skips the test unless a reachable Docker daemon is available.
func requireDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker binary not found in PATH; skipping Docker e2e test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "docker", "info").CombinedOutput(); err != nil {
		t.Skipf("docker daemon not reachable; skipping Docker e2e test: %v\n%s", err, string(out))
	}
}

// cli is a configured aerolab command runner bound to an isolated AEROLAB_HOME
// and the docker backend.
type cli struct {
	t   *testing.T
	bin string
	env []string
}

// newDockerCLI builds a runner: isolates AEROLAB_HOME to a temp dir, disables
// telemetry, sets the docker backend, and wires an optional features file.
func newDockerCLI(t *testing.T) *cli {
	t.Helper()
	requireDocker(t)
	bin := aerolabBinary(t)

	home := t.TempDir()
	env := append(os.Environ(),
		"AEROLAB_HOME="+home,
		"AEROLAB_TEST=1",
		"AEROLAB_TELEMETRY_DISABLE=1",
	)
	c := &cli{t: t, bin: bin, env: env}

	c.run("config", "backend", "-t", "docker")
	if ff := os.Getenv("AEROLAB_FEATURES_FILE"); ff != "" {
		c.run("config", "defaults", "-k", "*.FeaturesFilePath", "-v", ff)
	}
	return c
}

// newCloudCLI builds a runner configured for the AWS backend (which fronts the
// Aerospike Cloud commands). It isolates AEROLAB_HOME, disables telemetry, sets
// the aws backend + region, and wires an optional features file. It skips the
// test unless AEROLAB_E2E_CLOUD is set, since it needs real cloud credentials.
func newCloudCLI(t *testing.T) *cli {
	t.Helper()
	if os.Getenv("AEROLAB_E2E_CLOUD") == "" {
		t.Skip("set AEROLAB_E2E_CLOUD=1 (with valid AWS credentials) to run the Aerospike Cloud tier")
	}
	bin := aerolabBinary(t)
	region := getenvDefault("AEROLAB_E2E_AWS_REGION", "us-east-1")

	home := t.TempDir()
	env := append(os.Environ(),
		"AEROLAB_HOME="+home,
		"AEROLAB_TEST=1",
		"AEROLAB_TELEMETRY_DISABLE=1",
	)
	c := &cli{t: t, bin: bin, env: env}

	c.run("config", "backend", "-t", "aws", "-r", region, "-P", "eks")
	if ff := os.Getenv("AEROLAB_FEATURES_FILE"); ff != "" {
		c.run("config", "defaults", "-k", "*.FeaturesFilePath", "-v", ff)
	}
	return c
}

// run executes aerolab with the given args, failing the test on non-zero exit.
// It returns the combined stdout+stderr.
func (c *cli) run(args ...string) string {
	c.t.Helper()
	out, err := c.runErr(args...)
	if err != nil {
		c.t.Fatalf("aerolab %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

// runErr executes aerolab and returns the combined output and any error without
// failing the test, for cases where a non-zero exit is acceptable/expected.
func (c *cli) runErr(args ...string) (string, error) {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.bin, args...)
	cmd.Env = c.env
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// getenvDefault returns the environment value for key or def if unset/empty.
func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
