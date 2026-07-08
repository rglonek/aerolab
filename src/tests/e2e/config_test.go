//go:build integration_docker

package e2e_test

import (
	"strings"
	"testing"
)

// TestDockerVersion is a smoke test: the binary runs and reports a version.
func TestDockerVersion(t *testing.T) {
	c := newDockerCLI(t)
	out := c.run("version")
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected non-empty version output")
	}
}

// TestDockerConfigNetworks exercises the docker-only config network commands,
// mirroring the `config docker list-networks` / `prune-networks` steps in the
// legacy test.sh.
func TestDockerConfigNetworks(t *testing.T) {
	c := newDockerCLI(t)
	c.run("config", "docker", "list-networks")
	c.run("config", "docker", "prune-networks")
}

// TestDockerInventoryAndCleanup verifies the project cleanup + inventory listing
// commands run cleanly against the docker backend.
func TestDockerInventoryAndCleanup(t *testing.T) {
	c := newDockerCLI(t)
	c.run("inventory", "delete-project-resources", "-f")
	c.run("inventory", "list")
}
