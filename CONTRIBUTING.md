# Contributing to Aerolab

## Testing

Aerolab's tests are pure Go and split into tiers by cost and required
infrastructure. Full details, including the test-support package and every
environment variable, live in [`tests/README.md`](tests/README.md).

Quick reference (run from the repo root):

| Command | Tier | Needs |
|---------|------|-------|
| `make test` | unit + mock (hermetic) | nothing |
| `make test-cover` | unit + mock with coverage | nothing |
| `make test-docker` | Docker integration (`integration_docker`) | a running Docker daemon |
| `make test-cloud` | cloud integration (`integration_cloud`) | real AWS/GCP + Aerospike Cloud creds |

Guidelines:

- The default `make test` must stay hermetic: no network, Docker, cloud
  credentials, or machine-specific paths. New non-hermetic behavior must be
  guarded by an opt-in environment variable and skip by default.
- Tests requiring Docker go behind the `integration_docker` build tag and must
  skip cleanly when Docker is unavailable.
- Tests requiring real cloud credentials go behind the `integration_cloud`
  build tag and must skip unless their required environment variables are set.
- Use the `pkg/backend/backendtest` doubles (`FakeBackend`, `FakeCloud`,
  fixtures) rather than talking to real cloud SDKs in unit tests.
- Run `make test` (and, where relevant, `make test-docker`) before opening a PR.
