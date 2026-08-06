//go:build integration_aerospike_cloud

package e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file is the whole `integration_aerospike_cloud` tier: the `aerolab
// cloud ...` command surface, driven against the real Aerospike Cloud API.
//
// It is a separate tier from `integration_cloud` because it is a separate kind
// of run: it provisions billable managed clusters, takes tens of minutes, and
// exercises a service the AWS/GCP backend suites never touch. Run it with
// `make test-aerospike-cloud`, or narrow it to the cheap parts with
// `go test -tags=integration_aerospike_cloud -run 'Secrets|InstanceTypes' ./tests/e2e/`.
//
// Coverage of the command tree in cli/cmd/v1/cmdCloud.go:
//
//	list-instance-types              TestAerospikeCloudInstanceTypes
//	auth get-token                   TestAerospikeCloudAuthToken
//	gen-conf-templates               TestAerospikeCloudGenConfTemplates
//	secrets create/list/delete       TestAerospikeCloudSecretsLifecycle
//	clusters create/list/update/delete,
//	clusters get host/tls-cert, wait,
//	vpc-peering-status, enable-logs-access,
//	clusters credentials create/list/delete
//	                                 TestAerospikeCloudClusterLifecycle
//
// `clusters peer-vpc` is the one command with no direct coverage: `clusters
// create --vpc-id` runs the same peering stages as part of provisioning, and
// re-peering a live cluster into a second VPC would need a second VPC that the
// tier cannot assume exists. The peering it does perform is asserted from
// create's own stage reporting, not from `vpc-peering-status` -- see
// TestAerospikeCloudClusterLifecycle for why that command cannot be the judge.

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

type vpcPeeringStatus struct {
	ClusterID string `json:"clusterId"`
	Peerings  []struct {
		VpcID          string `json:"vpcId"`
		PeeringID      string `json:"peeringId"`
		CloudStatus    string `json:"cloudStatus"`
		Status         string `json:"status"`
		CompletedSteps int    `json:"completedSteps"`
		TotalSteps     int    `json:"totalSteps"`
		Steps          []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"steps"`
	} `json:"peerings"`
}

// TestAerospikeCloudInstanceTypes covers `cloud list-instance-types` in the
// shapes the CLI offers: the default table, machine-readable JSON, and the
// region filter.
func TestAerospikeCloudInstanceTypes(t *testing.T) {
	c := newAerospikeCloudCLI(t)
	region := aerospikeCloudRegion()

	c.run("cloud", "list-instance-types")

	// The command flattens the provider/region/type nesting the API returns into
	// one row per instance type, and the JSON renderer emits that flat slice, so
	// the keys are the Go field names of FlattenedInstanceType.
	var flattened []struct {
		Region string
		Type   string
		Arch   string
		Vcpus  int
		RAMGib int
	}
	out := c.run("cloud", "list-instance-types", "-o", "json")
	mustJSON(t, out, &flattened)
	if len(flattened) == 0 {
		t.Fatalf("expected at least one instance type in list-instance-types output:\n%s", out)
	}
	inRegion := 0
	for _, it := range flattened {
		if it.Type == "" || it.Region == "" {
			t.Errorf("instance type row is missing region or type: %+v", it)
		}
		if it.Region == region {
			inRegion++
		}
	}
	if inRegion == 0 {
		t.Errorf("expected at least one instance type in %s", region)
	}

	// --region filters the flattened rows, but only for the renderers that
	// consume them (the JSON and jq paths deliberately emit everything), so it
	// has to be asserted through one of those. `text` prints one labelled line
	// per row.
	out = c.run("cloud", "list-instance-types", "-o", "text", "--region", region)
	rows := 0
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "Region: ") {
			continue
		}
		rows++
		if !strings.HasPrefix(line, "Region: "+region+",") {
			t.Errorf("--region %s returned a row from another region: %s", region, line)
		}
	}
	if rows == 0 {
		t.Errorf("expected --region %s to return rows:\n%s", region, out)
	}
}

// TestAerospikeCloudAuthToken covers `cloud auth get-token`. The token is what
// every other cloud command depends on, so a failure here localizes an
// authentication problem instead of letting it surface as a confusing API
// error further down.
func TestAerospikeCloudAuthToken(t *testing.T) {
	c := newAerospikeCloudCLI(t)

	// The command prints the bare token to stdout when stdout is not a
	// terminal, which is the case here.
	out := strings.TrimSpace(c.run("cloud", "auth", "get-token"))
	if out == "" {
		t.Fatal("expected `cloud auth get-token` to print a token")
	}
	// A JWT, so at minimum it must not have come back as an error blob.
	if strings.Contains(out, " ") {
		t.Fatalf("expected a bare token, got:\n%s", out)
	}
}

// TestAerospikeCloudGenConfTemplates covers `cloud gen-conf-templates`, which
// downloads the published OpenAPI spec and writes the four request-body
// templates from it.
func TestAerospikeCloudGenConfTemplates(t *testing.T) {
	c := newAerospikeCloudCLI(t)

	dir := t.TempDir()
	c.run("cloud", "gen-conf-templates", "-d", dir)

	for _, name := range []string{
		"create-full.json",
		"create-aerospike-server.json",
		"update-full.json",
		"update-aerospike-server.json",
	} {
		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected template %s: %v", name, err)
		}
		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("template %s is not valid JSON: %v", name, err)
		}
	}
}

// TestAerospikeCloudSecretsLifecycle reproduces the secrets portion of
// test.sh's testcloud: create a secret, confirm it lists, then delete it.
func TestAerospikeCloudSecretsLifecycle(t *testing.T) {
	c := newAerospikeCloudCLI(t)

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

// TestAerospikeCloudClusterLifecycle walks a managed cluster through its whole
// life: create (which also peers the VPC and enables log access), inspect it,
// add and remove credentials, scale it, then delete it.
//
// `cloud clusters create` blocks until provisioning finishes and
// `cloud clusters delete` blocks until the cluster is decommissioned, so no
// separate wait step is needed around them.
func TestAerospikeCloudClusterLifecycle(t *testing.T) {
	c := newAerospikeCloudCLI(t)
	region := aerospikeCloudRegion()
	// "default" is aerolab's own default and resolves to the region's default
	// VPC, so the tier needs no VPC id configured to run; point it elsewhere
	// when the default VPC is not the one to peer into.
	vpc := getenvDefault("AEROLAB_ASCLOUD_VPC_ID", "default")
	instanceType := getenvDefault("AEROLAB_ASCLOUD_INSTANCE_TYPE", "m5d.large")
	updateInstanceType := getenvDefault("AEROLAB_ASCLOUD_UPDATE_INSTANCE_TYPE", "m5d.xlarge")
	const clusterName = "aerolabtest"

	// Ensure a clean slate and always clean up on exit.
	deleteClusterByName(t, c, clusterName)
	t.Cleanup(func() { deleteClusterByName(t, c, clusterName) })

	createOut := c.run("cloud", "clusters", "create", "-n", clusterName, "-i", instanceType,
		"-r", region, "--availability-zone-count=2", "--cluster-size=2",
		"--data-storage", "memory", "--vpc-id", vpc)

	// create drives VPC peering itself, and it is the authority on whether that
	// worked: it runs the four stages in order and returns an error the moment
	// one fails, which c.run above already turns into a test failure. What is
	// left to check is that they were not silently skipped, so look for each
	// stage having reported an outcome.
	//
	// Deliberately not asserted through `vpc-peering-status`: that command
	// re-derives each stage after the fact and cannot see enough to agree.
	// `Accept` is judged from the Aerospike Cloud API's cloudStatus, which still
	// reads "pending-acceptance" long after aerolab has accepted the connection
	// in AWS, and `AssociateDNS` is unverifiable because the private hosted zone
	// lives in the Aerospike Cloud account. Only OK steps count toward
	// completedSteps, so a perfectly good peering reports INCOMPLETE (2/4).
	for _, stage := range []string{"INITIATE", "ACCEPT", "ROUTE", "ASSOCIATE-DNS"} {
		// "Skipping" is a normal outcome -- an already-active peering, an
		// already-present route -- and means the stage ran and was satisfied.
		if !strings.Contains(createOut, "Stage "+stage+": Completed") &&
			!strings.Contains(createOut, "Stage "+stage+": Skipping") {
			t.Errorf("cluster create did not report VPC peering stage %s", stage)
		}
	}
	if !strings.Contains(createOut, "VPC peering setup completed successfully") {
		t.Error("cluster create did not report VPC peering as complete")
	}

	cid := clusterIDByName(t, c, clusterName)
	if cid == "" {
		t.Fatal("cluster creation failed: no id found")
	}

	// The cluster exists and is not on its way out. `wait` polls health.status,
	// so this also proves the status field the CLI reads is being populated.
	c.run("cloud", "clusters", "wait", "-c", cid, "--status-ne", "decommissioned",
		"--status-ne", "decommissioning", "-t", "600")

	// Connection details a client would need.
	if host := strings.TrimSpace(c.run("cloud", "clusters", "get", "host", "-c", cid)); host == "" {
		t.Error("expected `clusters get host` to print a host")
	}
	if cert := c.run("cloud", "clusters", "get", "tls-cert", "-c", cid); !strings.Contains(cert, "BEGIN CERTIFICATE") {
		t.Errorf("expected `clusters get tls-cert` to print a PEM certificate, got:\n%s", cert)
	}
	// Same lookup by name rather than id.
	if host := strings.TrimSpace(c.run("cloud", "clusters", "get", "host", "-n", clusterName)); host == "" {
		t.Error("expected `clusters get host -n` to print a host")
	}

	// `vpc-peering-status` is part of the command surface, so it has to run and
	// return a usable answer for the peering create just set up. Its per-step
	// verdicts are recorded rather than asserted, for the reasons above: the
	// only thing it can be held to is reporting the peering at all.
	var ps vpcPeeringStatus
	mustJSON(t, c.run("cloud", "clusters", "vpc-peering-status", "-c", cid), &ps)
	if len(ps.Peerings) == 0 {
		t.Error("expected `vpc-peering-status` to report a peering after cluster create")
	}
	for _, p := range ps.Peerings {
		if p.PeeringID == "" {
			t.Errorf("VPC peering for %s has no peering id: %+v", p.VpcID, p)
		}
		for _, s := range p.Steps {
			if s.Status == "FAILED" {
				t.Errorf("VPC peering %s step %s FAILED: %s", p.VpcID, s.Name, s.Error)
			}
		}
	}

	// The listing that operators actually read, including the VPC column that
	// needs the AWS backend to render.
	c.run("cloud", "clusters", "list", "--with-vpc-status")

	// create already granted log access to the current account; --append is the
	// incremental path, and with no --role it re-authorizes that same account
	// root ARN, so this needs no extra IAM setup.
	c.run("cloud", "clusters", "enable-logs-access", "-c", cid, "--append")

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

	c.run("cloud", "clusters", "update", "-c", cid, "--cluster-size", "4", "-i", updateInstanceType)
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
	if json.Unmarshal([]byte(firstJSONDocument(out)), &cl) != nil {
		return ""
	}
	for _, cluster := range cl.Clusters {
		if cluster.Name == name {
			return cluster.ID
		}
	}
	return ""
}

// mustJSON unmarshals the JSON document found in out into v, failing the test
// on error.
func mustJSON(t *testing.T, out string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(firstJSONDocument(out)), v); err != nil {
		t.Fatalf("failed to parse JSON output: %v\n%s", err, out)
	}
}

// firstJSONDocument extracts the JSON payload from command output, tolerating
// the log lines the CLI emits before and after it.
//
// It decodes rather than slicing between delimiters, because both ends of a
// slice are ambiguous here. Some commands print an object (`cloud secrets
// list`) and others a bare array (`cloud list-instance-types -o json`), so the
// delimiter to look for is not fixed; and the log lines themselves contain
// brackets -- every line carries a `[cloud.clusters.list]`-style prefix -- so
// the first '[' in the output is usually not the payload. Trying each '{' or
// '[' in turn and keeping the first that decodes as one complete value settles
// both, and stops at the end of that value so a trailing log line is ignored.
func firstJSONDocument(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] != '{' && s[i] != '[' {
			continue
		}
		var v json.RawMessage
		if err := json.NewDecoder(strings.NewReader(s[i:])).Decode(&v); err == nil {
			return string(v)
		}
	}
	// No JSON in the output: hand back the whole thing so the caller's failure
	// message shows what the command actually printed.
	return s
}
