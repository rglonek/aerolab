//go:build integration_docker

package e2e_test

import (
	"os"
	"strings"
	"testing"
)

// TestDockerTLSXDRData reproduces the TLS/XDR/data-manipulation portion of
// tests-to-add.sh against the Docker backend: stand up two clusters, generate
// and copy TLS certs, wire XDR between them, insert/delete data, and confirm
// asadm connectivity. Opt-in (heavy) and needs a features file.
func TestDockerTLSXDRData(t *testing.T) {
	requireExtended(t)
	c := newDockerCLI(t)
	asVer := getenvDefault("AEROLAB_E2E_ASVER", "8.*")

	dc1 := uniqueClusterName() + "a"
	dc2 := uniqueClusterName() + "b"
	t.Cleanup(func() {
		_, _ = c.runErr("cluster", "destroy", "-f", "-n", dc1+","+dc2)
	})

	c.run("inventory", "delete-project-resources", "-f")
	c.run("cluster", "create", "-c", "2", "-d", "ubuntu", "-i", "24.04", "-v", asVer, "-n", dc1)
	c.run("cluster", "create", "-c", "2", "-d", "ubuntu", "-i", "24.04", "-v", asVer, "-n", dc2)

	// TLS. Each run gets its own CA: `tls generate` reuses whatever CA it finds
	// in its work dir, so a shared one would have every run signing against the
	// first run's key.
	c.run("tls", "generate", "-n", dc1, "-W", t.TempDir())
	c.run("tls", "copy", "-s", dc1, "-d", dc2)

	// XDR.
	c.run("xdr", "connect", "-s", dc1, "-D", dc2)

	// `xdr connect` restarts the source cluster and returns as soon as the
	// restart is issued, so wait for the service to be up again before talking
	// to it.
	c.run("aerospike", "is-stable", "-n", dc1, "-w", "-o", "120", "-i")

	// Data insert/delete and connectivity.
	c.run("data", "insert", "-n", dc1, "-a", "1", "-z", "3000")
	c.run("data", "delete", "-n", dc1, "-a", "1", "-z", "1000")
	c.run("attach", "shell", "-n", dc2, "--", "asadm", "-e", "info")

	c.run("cluster", "destroy", "-f", "-n", dc1+","+dc2)
}

// TestDockerNetBlock reproduces the iptables net-block portion of
// tests-to-add.sh: block traffic between two clusters, list, then unblock.
func TestDockerNetBlock(t *testing.T) {
	requireExtended(t)
	c := newDockerCLI(t)
	asVer := getenvDefault("AEROLAB_E2E_ASVER", "8.*")

	dc1 := uniqueClusterName() + "a"
	dc2 := uniqueClusterName() + "b"
	t.Cleanup(func() {
		_, _ = c.runErr("cluster", "destroy", "-f", "-n", dc1+","+dc2)
	})

	c.run("inventory", "delete-project-resources", "-f")
	// `net block` drives iptables inside the container, which needs NET_ADMIN.
	// An unprivileged container fails with iptables exit status 4.
	c.run("cluster", "create", "-c", "1", "-d", "ubuntu", "-i", "24.04", "-v", asVer, "-n", dc1, "--docker.privileged")
	c.run("cluster", "create", "-c", "1", "-d", "ubuntu", "-i", "24.04", "-v", asVer, "-n", dc2, "--docker.privileged")

	c.run("net", "block", "-s", dc1, "-d", dc2)
	c.run("net", "list")
	c.run("net", "unblock", "-s", dc1, "-d", dc2)

	c.run("cluster", "destroy", "-f", "-n", dc1+","+dc2)
}

// TestDockerClientMatrix reproduces the client create/list/destroy matrix from
// tests-to-add.sh. The eksctl client is excluded because it requires external
// cloud credentials and a features file path.
func TestDockerClientMatrix(t *testing.T) {
	requireExtended(t)
	c := newDockerCLI(t)
	asVer := getenvDefault("AEROLAB_E2E_ASVER", "8.*")

	server := uniqueClusterName()
	clients := []string{"none", "base", "vscode", "ams", "tools", "graph"}
	t.Cleanup(func() {
		for _, name := range clients {
			_, _ = c.runErr("client", "destroy", "-f", "-n", name)
		}
		_, _ = c.runErr("cluster", "destroy", "-f", "-n", server)
	})

	c.run("inventory", "delete-project-resources", "-f")
	c.run("cluster", "create", "-v", asVer, "-n", server)

	c.run("client", "create", "none", "-n", "none")
	c.run("client", "create", "base", "-n", "base")
	c.run("client", "create", "vscode", "-n", "vscode")
	c.run("client", "create", "ams", "-n", "ams")
	c.run("client", "configure", "ams", "-n", "ams", "-s", server)
	c.run("client", "create", "tools", "-n", "tools")
	c.run("client", "configure", "tools", "-n", "tools", "-m", "ams")
	// -n names the client; the cluster to seed from comes from -C and defaults
	// to "asd", which is not the randomly named cluster created above.
	c.run("client", "create", "graph", "-n", "graph", "-C", server)

	list := c.run("client", "list")
	for _, name := range clients {
		if !strings.Contains(list, name) {
			t.Errorf("expected client %q in `client list` output:\n%s", name, list)
		}
	}

	for _, name := range clients {
		c.run("client", "destroy", "-f", "-n", name)
	}
}

// requireExtended skips unless the extended docker feature suite is explicitly
// enabled and a features file is configured for Enterprise images.
func requireExtended(t *testing.T) {
	t.Helper()
	if os.Getenv("AEROLAB_E2E_EXTENDED") == "" {
		t.Skip("set AEROLAB_E2E_EXTENDED=1 to run the extended docker feature tests")
	}
	if os.Getenv("AEROLAB_FEATURES_FILE") == "" {
		t.Skip("set AEROLAB_FEATURES_FILE to run the extended docker feature tests")
	}
}
