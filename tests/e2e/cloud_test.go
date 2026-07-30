//go:build integration_cloud

package e2e_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// cloud JSON response shapes (subset of fields we assert on). These mirror the
// Aerospike Cloud API collection schemas the CLI prints verbatim.
type secretsList struct {
	Secrets []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"secrets"`
}

type clustersList struct {
	Clusters []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"clusters"`
}

type credentialsList struct {
	Credentials []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"credentials"`
}

// TestCloudSecretsLifecycle reproduces the secrets portion of test.sh's
// testcloud: create a secret, confirm it lists, then delete it.
func TestCloudSecretsLifecycle(t *testing.T) {
	c := newCloudCLI(t)

	c.run("cloud", "list-instance-types")
	c.run("cloud", "secrets", "create", "--name", "aerolab", "--description", "aerolab", "--value", "aerolab")

	var sl secretsList
	mustJSON(t, c.run("cloud", "secrets", "list"), &sl)

	// Clean up every secret we own with our marker description, registered
	// before the assertion below so it still runs if the assertion fails.
	t.Cleanup(func() {
		for _, s := range sl.Secrets {
			if s.Description == "aerolab" {
				c.run("cloud", "secrets", "delete", "--secret-id", s.ID)
			}
		}
	})

	found := 0
	for _, s := range sl.Secrets {
		if s.Description == "aerolab" {
			found++
		}
	}
	if found < 1 {
		t.Fatalf("expected at least one 'aerolab' secret, got %d", found)
	}
}

// TestCloudClusterLifecycle reproduces the cluster + credentials portion of
// test.sh's testcloud: create a cluster, add/list/delete credentials, update
// the cluster, then delete it. It needs a VPC id (AEROLAB_E2E_VPC_ID).
//
// `cloud clusters create` blocks until provisioning finishes and
// `cloud clusters delete` blocks until the cluster is decommissioned, so no
// separate wait step is needed.
func TestCloudClusterLifecycle(t *testing.T) {
	c := newCloudCLI(t)
	vpc := os.Getenv("AEROLAB_E2E_VPC_ID")
	if vpc == "" {
		t.Skip("set AEROLAB_E2E_VPC_ID to run the cloud cluster lifecycle test")
	}
	region := getenvDefault("AEROLAB_E2E_AWS_REGION", "us-east-1")
	const clusterName = "aerolabtest"

	// Ensure a clean slate and always clean up on exit.
	deleteClusterByName(t, c, clusterName)
	t.Cleanup(func() { deleteClusterByName(t, c, clusterName) })

	c.run("cloud", "clusters", "create", "-n", clusterName, "-i", "m5d.large",
		"-r", region, "--availability-zone-count=2", "--cluster-size=2",
		"--data-storage", "memory", "--vpc-id", vpc)

	cid := clusterIDByName(t, c, clusterName)
	if cid == "" {
		t.Fatal("cluster creation failed: no id found")
	}

	// Passwords must be at least 8 characters.
	c.run("cloud", "clusters", "credentials", "create", "-c", cid,
		"--username", "aerolab1", "--password", "aerolab1", "--roles", "read-write", "--wait")
	c.run("cloud", "clusters", "credentials", "create", "-c", cid,
		"--username", "aerolab2", "--password", "aerolab2", "--roles", "read-write", "--wait")

	var cl credentialsList
	mustJSON(t, c.run("cloud", "clusters", "credentials", "list", "-c", cid), &cl)
	var del string
	for _, cred := range cl.Credentials {
		if cred.Name == "aerolab2" {
			del = cred.ID
		}
	}
	if del == "" {
		t.Fatal("expected credential 'aerolab2' to exist")
	}
	c.run("cloud", "clusters", "credentials", "delete", "-c", cid, "--credentials-id", del)

	c.run("cloud", "clusters", "update", "-c", cid, "--cluster-size", "4", "-i", "m5d.xlarge")
	c.run("cloud", "clusters", "delete", "-c", cid, "--force")
}

// deleteClusterByName deletes any cluster matching name, ignoring "not found".
func deleteClusterByName(t *testing.T, c *cli, name string) {
	t.Helper()
	if cid := clusterIDByName(t, c, name); cid != "" {
		_, _ = c.runErr("cloud", "clusters", "delete", "-c", cid, "--force")
	}
}

// clusterIDByName returns the id of the cluster with the given name, or "".
func clusterIDByName(t *testing.T, c *cli, name string) string {
	t.Helper()
	out, err := c.runErr("cloud", "clusters", "list", "-o", "json")
	if err != nil {
		return ""
	}
	var cl clustersList
	if json.Unmarshal([]byte(firstJSONObject(out)), &cl) != nil {
		return ""
	}
	for _, cluster := range cl.Clusters {
		if cluster.Name == name {
			return cluster.ID
		}
	}
	return ""
}

// mustJSON unmarshals the first JSON object found in out into v, failing the
// test on error.
func mustJSON(t *testing.T, out string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(firstJSONObject(out)), v); err != nil {
		t.Fatalf("failed to parse JSON output: %v\n%s", err, out)
	}
}

// firstJSONObject extracts the substring from the first '{' to the last '}',
// tolerating leading log lines the CLI may emit before the JSON payload.
func firstJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start < 0 || end < 0 || end < start {
		return s
	}
	return s[start : end+1]
}
