//go:build integration_cloud

package e2e_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// cloud JSON response shapes (subset of fields we assert on).
type secretsList struct {
	Secrets []struct {
		ID          string `json:"id"`
		Description string `json:"description"`
	} `json:"secrets"`
}

type databasesList struct {
	Databases []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"databases"`
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
	mustJSON(t, c.run("cloud", "secrets", "list", "-o", "json"), &sl)

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

// TestCloudDatabaseLifecycle reproduces the database + credentials portion of
// test.sh's testcloud: create a database, add/list/delete credentials, update
// the database, then delete it. It needs a VPC id (AEROLAB_E2E_VPC_ID).
func TestCloudDatabaseLifecycle(t *testing.T) {
	c := newCloudCLI(t)
	vpc := os.Getenv("AEROLAB_E2E_VPC_ID")
	if vpc == "" {
		t.Skip("set AEROLAB_E2E_VPC_ID to run the cloud database lifecycle test")
	}
	region := getenvDefault("AEROLAB_E2E_AWS_REGION", "us-east-1")
	const dbName = "aerolabtest"

	// Ensure a clean slate and always clean up on exit.
	deleteDBByName(t, c, dbName)
	t.Cleanup(func() { deleteDBByName(t, c, dbName) })

	c.run("cloud", "databases", "create", "-n", dbName, "-i", "m5d.large",
		"-r", region, "--availability-zone-count=2", "--cluster-size=2",
		"--data-storage", "memory", "--vpc-id", vpc)

	did := dbIDByName(t, c, dbName)
	if did == "" {
		t.Fatal("database creation failed: no id found")
	}

	c.run("cloud", "databases", "credentials", "create", "--database-id", did,
		"--username", "aerolab1", "--password", "aerolab1", "--privileges", "read-write", "--wait")
	c.run("cloud", "databases", "credentials", "create", "--database-id", did,
		"--username", "aerolab2", "--password", "aerolab2", "--privileges", "read-write", "--wait")

	var cl credentialsList
	mustJSON(t, c.run("cloud", "databases", "credentials", "list", "--database-id", did, "-o", "json"), &cl)
	var del string
	for _, cred := range cl.Credentials {
		if cred.Name == "aerolab2" {
			del = cred.ID
		}
	}
	if del == "" {
		t.Fatal("expected credential 'aerolab2' to exist")
	}
	c.run("cloud", "databases", "credentials", "delete", "--database-id", did, "--credentials-id", del)

	c.run("cloud", "databases", "update", "--database-id", did, "--cluster-size", "4", "-i", "m5d.xlarge")
	c.run("cloud", "databases", "delete", "--database-id", did, "--force", "--wait")
}

// deleteDBByName deletes any database matching name, ignoring "not found".
func deleteDBByName(t *testing.T, c *cli, name string) {
	t.Helper()
	if did := dbIDByName(t, c, name); did != "" {
		_, _ = c.runErr("cloud", "databases", "delete", "--database-id", did, "--force", "--wait")
	}
}

// dbIDByName returns the id of the database with the given name, or "".
func dbIDByName(t *testing.T, c *cli, name string) string {
	t.Helper()
	out, err := c.runErr("cloud", "databases", "list", "-o", "json")
	if err != nil {
		return ""
	}
	var dl databasesList
	if json.Unmarshal([]byte(firstJSONObject(out)), &dl) != nil {
		return ""
	}
	for _, d := range dl.Databases {
		if d.Name == name {
			return d.ID
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
