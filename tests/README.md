# Aerolab Testing

Aerolab's tests are pure Go (`go test`) and are split into tiers by cost and
required infrastructure. There are no bash test harnesses; everything runs
through `go test` and the `Makefile` targets.

## Tiers at a glance

| Tier | Build tag | What it needs | Make target | Runs in CI |
|------|-----------|---------------|-------------|------------|
| Unit + mock | _(none)_ | nothing (hermetic) | `make test` | every PR |
| Docker integration | `integration_docker` | a running Docker daemon | `make test-docker` | pushes / manual |
| Cloud integration | `integration_cloud` | real AWS/GCP creds | `make test-cloud` | opt-in / manual |
| Aerospike Cloud | `integration_aerospike_cloud` | Aerospike Cloud API creds + AWS | `make test-aerospike-cloud` | opt-in / manual |

All targets run from the repo root (the Go module root) and export
`GOWORK=off` / `GOFLAGS=-mod=vendor`. Each has a `-nogen` variant
(`make test-nogen`, `make test-docker-nogen`, `make test-cloud-nogen`,
`make test-aerospike-cloud-nogen`) that skips the `go generate` step; see
[Running every tier](#running-every-tier).

## Unit + mock tests (default)

```sh
make test        # -race -shuffle=on, hermetic
make test-cover   # + coverage.out summary
```

These must stay hermetic: no network, no Docker, no cloud credentials, no
machine-specific paths. A few historically non-hermetic tests are guarded by
opt-in environment variables and skip by default:

- `AEROLAB_TEST_GRAFANA=1` — `pkg/agi/grafanafix` (needs a local Grafana).
- `AEROLAB_TEST_INGEST_SOURCES=1` — `pkg/agi/ingest` (downloads real log archives).

### Test-support package

`pkg/backend/backendtest` provides reusable doubles and fixtures:

- `FakeBackend` / `FakeCloud` — programmable `backends.Backend` / `backends.Cloud`
  implementations that record calls and inject errors.
- `NewInstance` / `NewCluster` / `NewInventory` (+ `With*` options) — fixtures.
- `RegisterFakeCloud(t, backendType, fake)` — registers a fake in the global
  backend registry and restores it via `t.Cleanup`. Tests using it must **not**
  call `t.Parallel()` (the registry is process-global).
- `QuietLogger()` — a near-silent logger for clean test output.

CLI command logic is unit-tested by injecting a fake backend through the
test-only `Init.BackendOverride` seam in `cli/cmd/v1/initialize.go`.

## Docker integration tests (`integration_docker`)

```sh
make test-docker
# or
GOWORK=off GOFLAGS=-mod=vendor go test -tags=integration_docker ./tests/... 
```

Covers `tests/e2e` (CLI end-to-end lifecycle) and `tests/installers` (install
scripts run inside containers). Tests skip automatically when Docker is missing
or the daemon is unreachable.

### Architecture

Both suites test the **host architecture only**: an arm64 machine installs
aarch64 packages into arm64 containers, and an amd64 machine installs x86_64
packages into amd64 containers. Nothing runs under emulation, because a package
set that would never be deployed on that host is not worth testing.

`tests/installers` matrices spell images with an `{arch}` placeholder — for
example `{arch}/rockylinux:9` or `quay.io/centos/{arch}:stream8` — which
`installertest.Image` resolves to the per-architecture registry namespace
(`amd64` or `arm64v8`). `installertest.RunScript` always passes
`docker run --platform`, which is required rather than cosmetic: without it
Docker resolves a multi-arch tag against the host platform and exits 125 on
manifest lists that have no entry for it.

A matrix entry whose distro has no upstream build for the host architecture is
logged and skipped. When a whole release line ships nothing for the host
architecture (anything predating arm64 support, such as server 4.x and 5.x) the
test skips; if the line does ship the architecture but no entry matched, that is
a stale distro list and it fails.

Both make targets pass `-count=1`. These suites talk to Docker and cloud APIs
that the go test cache cannot observe, so without it a re-run replays a stale
cached PASS without touching any infrastructure.

## Cloud integration tests (`integration_cloud`)

```sh
make test-cloud
```

Covers `tests/backend` (AWS/GCP backend behavior) and the `tests/e2e` cloud
tier. These require real credentials and are opt-in; they skip unless the
relevant environment variables are set. The `aerolab cloud ...` commands are
**not** part of this tier — see
[Aerospike Cloud tests](#aerospike-cloud-tests-integration_aerospike_cloud).

If you invoke `go test` directly instead of using the make target, pass
`-tags=integration_cloud,embedexpiry`. A `-tags` flag on the command line
replaces the one in `GOFLAGS` rather than adding to it, so omitting
`embedexpiry` builds the expiry stub and every expiry test fails with
"this build does not include the embedded expiry binary".

Note that `tests/backend` is destructive: it terminates instances and deletes
volumes, firewalls, in-account images, and expiry systems in the configured
regions. Point it at a throwaway account/project.

### `tests/backend` environment variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `AEROLAB_CLOUD` | `aws`, `gcp`, `docker`, or `podman` | _(required)_ |
| `AEROLAB_<CLOUD>_TEST_REGIONS` | comma-separated regions, e.g. `AEROLAB_GCP_TEST_REGIONS=us-central1` | _(required)_ |
| `AWS_PROFILE` | AWS shared-credentials profile | _(required for aws)_ |
| `GCP_PROJECT` | GCP project id | _(required for gcp)_ |
| `AEROLAB_GCP_AUTH_METHOD` | `any`, `login`, or `service-account` | `service-account` |
| `AEROLAB_GCP_CLIENT_ID` | OAuth client id; required by `login` | _(unset)_ |
| `AEROLAB_GCP_CLIENT_SECRET` | OAuth client secret; empty for the PKCE flow | _(unset)_ |
| `AEROLAB_GCP_AUTO_ENABLE_SERVICES` | let the backend enable the Google Cloud services it needs | on |
| `AEROLAB_GCP_USE_IAP` | route all SSH/SFTP through IAP TCP forwarding | off |
| `AEROLAB_GCP_NO_PUBLIC_IP` | create every GCP instance without a public IP | off |
| `AEROLAB_SKIP_CLEANUP` | leave created resources behind after the run | off |
| `AEROLAB_TEST_CUSTOM_TMPDIR` | fixed temp dir instead of a fresh one | fresh `mkdtemp` |
| `AEROLAB_TEST_DNS_DOMAIN` | domain of a hosted zone you own, e.g. `example.com` | _(unset → DNS test skips)_ |
| `AEROLAB_TEST_DNS_ZONE_ID` | Route53 hosted zone id (aws) / managed zone name (gcp) | _(unset → DNS test skips)_ |
| `AEROLAB_TEST_DNS_REGION` | Route53 region | `us-east-1` (aws), `global` (gcp) |
| `AEROLAB_TEST_AWS_INSTANCE_TYPE` | x86 instance type used by the arch test | `r6a.large` |
| `AEROLAB_TEST_AWS_ARM_INSTANCE_TYPE` | arm64 instance type used by the arch test | `r6g.large` |
| `AEROLAB_TEST_GCP_INSTANCE_TYPE` | x86 machine type used by the arch test | `e2-standard-4` |
| `AEROLAB_TEST_GCP_ARM_INSTANCE_TYPE` | arm64 machine type used by the arch test | `t2a-standard-4` |

GCP authenticates through Application Default Credentials by default, the same
as the CLI, so `gcloud auth application-default login` (or a service account on
the instance) is all the suite needs. `AEROLAB_GCP_AUTH_METHOD=login` runs the
interactive browser OAuth flow instead and only works with an OAuth client id
in `AEROLAB_GCP_CLIENT_ID`; aerolab ships no built-in client id.

Set `AEROLAB_GCP_USE_IAP=1` and `AEROLAB_GCP_NO_PUBLIC_IP=1` when the target
project only reaches instances that way, so the suite exercises the same path
your environment actually uses. The two are independent: IAP is never
auto-enabled just because public IPs are disabled. A private-only project needs
Cloud NAT for instance egress, and IAP needs `iap.googleapis.com` enabled plus
the `roles/iap.tunnelResourceAccessor` permission. The suite enables the
services it needs itself (the run is non-interactive, so the alternative is a
hard failure at setup); set `AEROLAB_GCP_AUTO_ENABLE_SERVICES=0` to enable them
by hand instead.

Pinning `AEROLAB_TEST_CUSTOM_TMPDIR` is worthwhile on GCP: the OAuth token is
cached at `<tmpdir>/gcp_token.json`, so a fixed directory avoids a browser
login on every run.

### The DNS test

`Test20_InstancesDNS` reads the hosted zone directly, through Route53 and Cloud
DNS, rather than trusting the inventory: both backends create and delete their
records on a best-effort basis (a failure is logged and the create or terminate
still succeeds), so a run that never touched the zone would otherwise pass. It
asserts that create put an A record per instance in the zone pointing at the
instance's routable address, that `CleanupDNS()` deletes a planted record whose
instance no longer exists while leaving the running instances' records alone,
and that terminate removed the records again.

Consequently the configured credentials need to list, create, and delete records
in the zone, plus tag it (`route53:ChangeTagsForResource` on AWS, and on GCP
`dns.managedZones.update`, since each backend marks the zone as its own and
`CleanupDNS()` only touches zones bearing that mark). On GCP the managed zone
must live in `GCP_PROJECT`, and Cloud DNS (`dns.googleapis.com`) has to be
enabled — the suite's `AEROLAB_GCP_AUTO_ENABLE_SERVICES` does not cover it.
The test plants its one synthetic record under the zone's own name using an
address from the RFC 5737 documentation range, and deletes it again.

## Aerospike Cloud tests (`integration_aerospike_cloud`)

```sh
make test-aerospike-cloud
```

Covers the `aerolab cloud ...` command tree — the Aerospike Cloud managed
service — plus the AWS calls those commands make on their own behalf (VPC
peering and S3 log access). It lives in `tests/e2e/aerospikecloud_test.go`.

This is its own tier rather than a switch inside `integration_cloud` because it
is its own kind of run: it provisions **billable** managed clusters, takes tens
of minutes, and targets a service the AWS/GCP backend suites never touch. Being
separately tagged means it can be started, re-run, and read on its own.

Two credential sets have to be in place:

- `AEROSPIKE_CLOUD_KEY` / `AEROSPIKE_CLOUD_SECRET` — the Aerospike Cloud API
  credentials aerolab itself reads. Every test in the tier skips without them.
- AWS credentials for the region under test, since `clusters create` peers the
  cluster's VPC and grants the account access to the cluster's log bucket.

The default region is `us-west-2`, deliberately not the AWS tier's. The two
tiers share an AWS account, and the AWS backend suite is destructive within the
regions it is pointed at, so keeping them apart means neither can reach the
other's resources.

To iterate on the cheap commands without waiting for a cluster, narrow by name:

```sh
GOWORK=off GOFLAGS=-mod=vendor go test -tags=integration_aerospike_cloud,embedexpiry \
  -count=1 -v -run 'Secrets|InstanceTypes|AuthToken|GenConfTemplates' ./tests/e2e/
```

### `tests/e2e` Aerospike Cloud environment variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `AEROSPIKE_CLOUD_KEY` | Aerospike Cloud API key | _(unset → tier skips)_ |
| `AEROSPIKE_CLOUD_SECRET` | Aerospike Cloud API secret | _(unset → tier skips)_ |
| `AEROSPIKE_CLOUD_ENV` | Set to `dev` to target the dev control plane | production |
| `AEROLAB_ASCLOUD_REGION` | Region the managed cluster is created in | `us-west-2` |
| `AEROLAB_ASCLOUD_AWS_PROFILE` | AWS profile for the peering / log access calls | `AWS_PROFILE`, else the default profile |
| `AEROLAB_ASCLOUD_VPC_ID` | VPC to peer the cluster into | `default` (aerolab resolves the region's default VPC) |
| `AEROLAB_ASCLOUD_INSTANCE_TYPE` | Instance type to create with | `m5d.large` |
| `AEROLAB_ASCLOUD_UPDATE_INSTANCE_TYPE` | Instance type `clusters update` scales to | `m5d.xlarge` |
| `AEROLAB_ASCLOUD_CMD_TIMEOUT` | Per-invocation budget for one `aerolab` command | `90m` |

### Timeouts

The managed service sets the pace, and two budgets have to cover it: the
per-invocation one above, and the `-timeout` on the make target (240m). Both are
deliberately generous. `clusters create` provisions a cluster and then peers its
VPC — its own progress message says "up to an hour" — and `clusters delete`
blocks until the cluster is decommissioned.

Exceeding either budget kills a subprocess or the whole test binary, which
reports as `signal: killed` with no error from aerolab, and worse, skips the
cleanup: a cluster killed mid-create keeps running (billable) and leaves behind
the blackhole route create installed in the VPC to reserve its CIDR. The next
run reaps the cluster, since the test deletes any pre-existing `aerolabtest`
before creating, but nothing reaps the route — check the VPC's route tables for
a blackhole entry on `10.130.0.0/19` (or the next free /19) if a run was killed.

`clusters peer-vpc` is the one command in the tree without direct coverage:
`clusters create --vpc-id` runs the same peering stages during provisioning, and
re-peering a live cluster would need a second VPC the tier cannot assume exists.
The peering that create performs is asserted from create's own stage reporting
(`Stage INITIATE/ACCEPT/ROUTE/ASSOCIATE-DNS: Completed`), which is the authority
on it: create runs the stages and fails the moment one does. `vpc-peering-status`
is still exercised, but its verdict is recorded rather than asserted — it
re-derives each stage afterwards and cannot see enough to agree. It judges
`Accept` from the Aerospike Cloud API's `cloudStatus`, which still reads
`pending-acceptance` long after aerolab has accepted the connection in AWS, and
it cannot verify `AssociateDNS` at all because the private hosted zone lives in
the Aerospike Cloud account. Only `OK` steps count toward `completedSteps`, so a
healthy peering reports `INCOMPLETE (2/4)`.

Every aerolab invocation is logged with its output and duration, pass or fail.
The command that explains a failure is usually an earlier one that succeeded, and
at an hour and a billable cluster per run, re-running to recover its output is
not a reasonable ask. This makes the docker tier's log substantially larger.

## `tests/e2e` environment variables

The e2e suite drives the real `aerolab` binary. No machine-specific paths are
hardcoded; everything is configured via environment. Give every path below as an
absolute path: the harness runs aerolab from a per-test working directory so
that nothing it writes to a relative path (the `tls generate` CA in particular)
lands in the source tree or collides with a concurrently running tier.

| Variable | Purpose | Default |
|----------|---------|---------|
| `AEROLAB_BIN` | Use a prebuilt aerolab binary instead of building one | build from source |
| `AEROLAB_FEATURES_FILE` | Aerospike features file (Enterprise images need it) | _(unset → lifecycle tests skip)_ |
| `AEROLAB_E2E_DISTRO` | Base distro for the lifecycle test | `ubuntu` |
| `AEROLAB_E2E_DISTRO_VER` | Distro version | `24.04` |
| `AEROLAB_E2E_ASVER` | Aerospike version selector | `8.*` |
| `AEROLAB_E2E_OS_MATRIX` | Run the multi-distro OS matrix test | off |
| `AEROLAB_E2E_EXTENDED` | Run the extended docker suite (TLS/XDR/data/net/clients) | off |
| `AEROLAB_E2E_AWS_REGION` | AWS region for the cloud tier | `us-east-1` |
| `AEROLAB_E2E_AWS_PROFILE` | AWS shared-credentials profile for the cloud tier | `AWS_PROFILE`, else the default profile |
| `AEROLAB_E2E_MIGRATE` | Run the `inventory migrate --dry-run` test | off |
| `AEROLAB_E2E_SSH_KEY_PATH` | SSH key path passed to `inventory migrate` | _(optional)_ |

Each entry in the OS matrix records the Aerospike release range that ships
packages for that distro, and skips unless `AEROLAB_E2E_ASVER` provably falls
inside it. The default `8.*` selector is deliberately vague, so it skips the
distros at either edge of the range (Ubuntu 26.04, which needs 8.1.3+, and
Ubuntu 20.04 / Debian 11, which were dropped in 8.0). Pin `AEROLAB_E2E_ASVER` to
a concrete version to exercise those.

Example — full Docker golden path against a local features file:

```sh
AEROLAB_FEATURES_FILE=/path/to/features.conf \
  go test -tags=integration_docker -run TestDockerClusterLifecycle -v ./tests/e2e/
```

## Running every tier

`tests/everything.sh` runs every tier with a single set of variables; edit the
values at the top to match your environment. It writes one log per tier to
`tests/log/` (`generate.log`, `hermetic.log`, `docker.log`, `aws.log`,
`gcp.log`, `ascloud.log`, each bracketed by a timestamp and ending with the
tier's exit code). Those logs are tracked in git, so commit them along with the
changes they exercised. The script exits with the sum of the tier exit codes.

The `ascloud` tier (Aerospike Cloud) comes last and is dropped from the run
entirely — with a line saying so — when `AEROSPIKE_CLOUD_KEY` /
`AEROSPIKE_CLOUD_SECRET` are unset. Every test in it would skip without them,
and a tier that provisioned and verified nothing should not report green.

The tiers run **concurrently**. They spend nearly all of their wall time waiting
on Docker or a cloud API, and they touch disjoint resources: separate backends,
separate `AEROLAB_TEST_CUSTOM_TMPDIR`s, a per-test `AEROLAB_HOME`, and a
per-test working directory for the aerolab subprocess. That last one matters
because `tls generate` writes its CA — private key included — to a path relative
to the working directory and reuses whatever CA it finds there; the docker and
cloud tiers run the same `tests/e2e` package from separate processes, so a
shared working directory would have them generating over each other's CA. The
`aws` and `ascloud` tiers share an AWS account but are pointed at different
regions, so the AWS suite's destructive cleanup cannot reach the managed
cluster's peering.

Set `AEROLAB_TEST_SEQUENTIAL=1` to run one tier at a time instead, which is
easier to follow when a tier is misbehaving. Note that in either mode a failing
hermetic tier no longer aborts the run; every tier reports its own exit code.

`go generate` runs once, up front, rather than as a prerequisite of each tier.
It rewrites shared embed artifacts in place (deleting and recreating
`expiry.linux.amd64.zip` and `pkg/webui/dist`), so concurrent copies would race
on those files, and its npm build would otherwise be repeated once per tier. The
make targets come in two forms for this reason: `make test-docker` generates
first and is self-contained, while `make test-docker-nogen` skips generate and
assumes the caller has already run it. If generate fails, the script stops
before starting any tier.

## Adding tests

- Prefer hermetic unit tests. Use `pkg/backend/backendtest` doubles instead of
  real cloud SDKs or Docker.
- Anything that needs Docker goes behind `//go:build integration_docker` and
  must skip cleanly when Docker is unavailable.
- Anything that needs real cloud credentials goes behind
  `//go:build integration_cloud` and must skip unless its required env is set.
- Anything that drives the Aerospike Cloud API goes behind
  `//go:build integration_aerospike_cloud` instead, so a run that only wants
  AWS/GCP backend coverage cannot provision a billable managed cluster.
- Never hardcode machine-specific paths, credentials, or account/VPC ids — read
  them from environment variables and skip when absent.
