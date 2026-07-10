//go:build integration_docker

package e2e_test

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// uniqueClusterName returns a short, docker-safe cluster name unique per run.
func uniqueClusterName() string {
	return "e2e" + strconv.FormatInt(time.Now().UnixNano()%100000, 10)
}

// TestDockerClusterLifecycle reproduces the core Docker lifecycle from
// test.sh's runtest: create -> list -> aerospike stop/start -> is-stable ->
// attach exec -> cluster stop/start -> destroy -> verify gone.
//
// It requires an Aerospike features file (AEROLAB_FEATURES_FILE) because the
// default server images are Enterprise and will not start without one. When the
// features file is not configured the test skips rather than fails.
func TestDockerClusterLifecycle(t *testing.T) {
	if os.Getenv("AEROLAB_FEATURES_FILE") == "" {
		t.Skip("set AEROLAB_FEATURES_FILE to run the Docker cluster lifecycle test (Enterprise images require a feature key)")
	}
	c := newDockerCLI(t)

	distro := getenvDefault("AEROLAB_E2E_DISTRO", "ubuntu")
	distroVer := getenvDefault("AEROLAB_E2E_DISTRO_VER", "24.04")
	asVer := getenvDefault("AEROLAB_E2E_ASVER", "8.*")

	name := uniqueClusterName()

	// Always attempt to clean up the cluster, even on failure.
	t.Cleanup(func() {
		_, _ = c.runErr("cluster", "destroy", "-n", name, "--force")
	})

	// Fresh start.
	c.run("inventory", "delete-project-resources", "-f")

	// Create a small 2-node cluster.
	c.run("cluster", "create", "-n", name, "-c", "2", "-d", distro, "-i", distroVer, "-v", asVer)

	// It should show up in the listing.
	list := c.run("cluster", "list")
	if !strings.Contains(list, name) {
		t.Fatalf("expected cluster %q in `cluster list` output:\n%s", name, list)
	}

	// Aerospike service lifecycle + stability wait.
	c.run("aerospike", "stop", "-n", name)
	c.run("aerospike", "start", "-n", name)
	c.run("aerospike", "is-stable", "-n", name, "-w", "-o", "60", "-i")

	// Remote command execution across all nodes.
	c.run("attach", "shell", "-n", name, "-l", "all", "--", "ls", "/tmp")

	// Container stop/start cycle. Restarted containers must re-establish SSH;
	// the backend refreshes the (previously empty, because the container was
	// stopped) published-port mapping before probing ssh-readiness.
	c.run("cluster", "stop", "-n", name)
	c.run("cluster", "start", "-n", name)
	c.run("aerospike", "is-stable", "-n", name, "-w", "-o", "60", "-i")

	// Partial node operations.
	c.run("cluster", "stop", "-n", name, "-l", "1")
	c.run("cluster", "start", "-n", name)

	// Destroy and confirm it is gone.
	c.run("cluster", "destroy", "-n", name, "--force")
	after := c.run("cluster", "list")
	if strings.Contains(after, name) {
		t.Fatalf("cluster %q still present after destroy:\n%s", name, after)
	}
}
