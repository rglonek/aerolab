# Aerolab Testing

Aerolab's tests are pure Go (`go test`) and are split into tiers by cost and
required infrastructure. There are no bash test harnesses; everything runs
through `go test` and the `Makefile` targets.

## Tiers at a glance

| Tier | Build tag | What it needs | Make target | Runs in CI |
|------|-----------|---------------|-------------|------------|
| Unit + mock | _(none)_ | nothing (hermetic) | `make test` | every PR |
| Docker integration | `integration_docker` | a running Docker daemon | `make test-docker` | pushes / manual |
| Cloud integration | `integration_cloud` | real AWS/GCP creds + Aerospike Cloud | `make test-cloud` | opt-in / manual |

All targets run from the repo root (the Go module root) and export
`GOWORK=off` / `GOFLAGS=-mod=vendor`.

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

## Cloud integration tests (`integration_cloud`)

```sh
make test-cloud
```

Covers `tests/backend` (AWS/GCP backend behavior) and the `tests/e2e` cloud
tier. These require real credentials and are opt-in; they skip unless the
relevant environment variables are set.

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

Set `AEROLAB_GCP_USE_IAP=1` and `AEROLAB_GCP_NO_PUBLIC_IP=1` when the target
project only reaches instances that way, so the suite exercises the same path
your environment actually uses. The two are independent: IAP is never
auto-enabled just because public IPs are disabled. A private-only project needs
Cloud NAT for instance egress, and IAP needs `iap.googleapis.com` enabled plus
the `roles/iap.tunnelResourceAccessor` permission.

Pinning `AEROLAB_TEST_CUSTOM_TMPDIR` is worthwhile on GCP: the OAuth token is
cached at `<tmpdir>/gcp_token.json`, so a fixed directory avoids a browser
login on every run.

## `tests/e2e` environment variables

The e2e suite drives the real `aerolab` binary. No machine-specific paths are
hardcoded; everything is configured via environment.

| Variable | Purpose | Default |
|----------|---------|---------|
| `AEROLAB_BIN` | Use a prebuilt aerolab binary instead of building one | build from source |
| `AEROLAB_FEATURES_FILE` | Aerospike features file (Enterprise images need it) | _(unset → lifecycle tests skip)_ |
| `AEROLAB_E2E_DISTRO` | Base distro for the lifecycle test | `ubuntu` |
| `AEROLAB_E2E_DISTRO_VER` | Distro version | `24.04` |
| `AEROLAB_E2E_ASVER` | Aerospike version selector | `8.*` |
| `AEROLAB_E2E_OS_MATRIX` | Run the multi-distro OS matrix test | off |
| `AEROLAB_E2E_EXTENDED` | Run the extended docker suite (TLS/XDR/data/net/clients) | off |
| `AEROLAB_E2E_CLOUD` | Enable the Aerospike Cloud tier | off |
| `AEROLAB_E2E_AWS_REGION` | AWS region for the cloud tier | `us-east-1` |
| `AEROLAB_E2E_AWS_PROFILE` | AWS shared-credentials profile for the cloud tier | `AWS_PROFILE`, else the default profile |
| `AEROLAB_E2E_VPC_ID` | VPC id used when creating a cloud cluster | _(unset → cluster test skips)_ |
| `AEROLAB_E2E_MIGRATE` | Run the `inventory migrate --dry-run` test | off |
| `AEROLAB_E2E_SSH_KEY_PATH` | SSH key path passed to `inventory migrate` | _(optional)_ |

Example — full Docker golden path against a local features file:

```sh
AEROLAB_FEATURES_FILE=/path/to/features.conf \
  go test -tags=integration_docker -run TestDockerClusterLifecycle -v ./tests/e2e/
```

`tests/everything.sh` runs every tier in order with a single set of variables;
edit the values at the top to match your environment.

## Adding tests

- Prefer hermetic unit tests. Use `pkg/backend/backendtest` doubles instead of
  real cloud SDKs or Docker.
- Anything that needs Docker goes behind `//go:build integration_docker` and
  must skip cleanly when Docker is unavailable.
- Anything that needs real cloud credentials goes behind
  `//go:build integration_cloud` and must skip unless its required env is set.
- Never hardcode machine-specific paths, credentials, or account/VPC ids — read
  them from environment variables and skip when absent.
