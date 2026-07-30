# Getting Started with GCP Backend

The GCP backend creates and manages Aerospike clusters on Compute Engine. Use it for
production-like testing and performance benchmarks.

## Prerequisites

1. An active Google Cloud Platform account and project
2. The [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) installed
3. Authenticate with Application Default Credentials (ADC) — Aerolab uses these, not your
   `gcloud` login, to talk to GCP:

```bash
gcloud auth application-default login
# or, to set the project at the same time:
gcloud auth application-default login --project=your-project-id
```

4. **Required GCP APIs**: Aerolab needs Compute Engine, Service Usage, Cloud Billing, IAP,
   and (for expiry automation) the Cloud Functions stack. Aerolab can enable what's missing
   for you — see [GCP Services (APIs) Required by AeroLab](gcp-services.md) for the full
   list and the `--gcp.auto-enable-services` flag.

## Gotchas

### `--gcp.no-public-ip` and `--gcp.use-iap` are independent

`--gcp.use-iap` is the **sole** trigger for routing SSH/SFTP through IAP. Aerolab does
**not** auto-route through IAP just because public IPs are disabled — you must opt into
each independently:

| `--gcp.no-public-ip` | `--gcp.use-iap` | Behaviour |
| --- | --- | --- |
| no | no | Default: instances get public IPs; SSH dials the public IP. |
| yes | no | No public IP; SSH attempts the private IP and fails unless you have VPN/peering. |
| no | yes | Instances still get public IPs, but SSH/SFTP routes through IAP anyway. |
| yes | yes | Canonical no-public-IP, IAP-only deployment. |

### Optional: Route SSH/SFTP through IAP

For deployments where your desktop can't reach VM IPs directly (no public IPs, no
VPN/peering), route SSH/SFTP through Google [Identity-Aware Proxy TCP forwarding](https://cloud.google.com/iap/docs/using-tcp-forwarding):

```bash
aerolab config backend -t gcp -r us-central1 -o your-project-id \
  --gcp.no-public-ip --gcp.use-iap
```

**One-time prerequisites per project**, none of which Aerolab does for you except the first:

1. **Enable the IAP API.** Aerolab enables `iap.googleapis.com` automatically the first time
   you run `config backend ... --gcp.use-iap` (prompting first if interactive, silently if
   `--gcp.auto-enable-services` is set). If your principal can't enable APIs, ask an owner to
   run `gcloud services enable iap.googleapis.com --project=your-project-id`.
2. **Grant the IAM role.** You must do this yourself — Aerolab does not:
   ```bash
   gcloud projects add-iam-policy-binding your-project-id \
     --member=user:you@example.com \
     --role=roles/iap.tunnelResourceAccessor
   ```
3. **Firewall.** IAP connects from the range `35.235.240.0/20`. Aerolab's default
   `aerolab-default` rule already allows `tcp:22` from `0.0.0.0/0`, which covers it — but
   **if you use a locked-down custom firewall, you must add this range yourself**; Aerolab
   does not add it automatically.

If `attach shell` reports a 403 from `tunnel.cloudproxy.app`, recheck step 2. If
`config backend` itself fails to enable the API, recheck that your principal has
`roles/serviceusage.serviceUsageAdmin` (step 1).

### Cloud NAT is required for `--gcp.no-public-ip`

Instances without a public IP have no outbound internet by default, but the install script
needs to reach `download.aerospike.com` and distro package mirrors. To catch this early,
Aerolab checks for a Cloud NAT covering the target subnet **before** every `cluster create`
and aborts if none is found:

```bash
gcloud compute routers create aerolab-router \
  --network=default --region=us-central1 --project=your-project-id

gcloud compute routers nats create aerolab-nat \
  --router=aerolab-router --region=us-central1 \
  --auto-allocate-nat-external-ips --nat-all-subnet-ip-ranges \
  --enable-logging --project=your-project-id
```

Two caveats worth knowing:

- Egress from a VPN, VPC peering, an internal proxy, or a hand-rolled NAT VM is invisible to
  this check (it only looks at `compute.routers.list`). Bypass it with
  `AEROLAB_SKIP_NAT_CHECK=1` — see the
  [environment variables reference](../reference/environment-variables.md#aerolab_skip_nat_check).
- If the check itself can't run (e.g. your principal lacks `compute.routers.list`
  permission, or a transient API error), Aerolab **fails open** and lets the create proceed
  without confirming NAT actually exists. Don't rely on the check as your only signal that
  egress is set up correctly.

### Flags don't persist across `config backend` runs

`--gcp.no-public-ip`, `--gcp.use-iap`, `--gcp.auto-enable-services`, and `--skip-pricing` are
**not** merged with your last saved config — omitting a previously-set flag on a later
`config backend` call silently turns it back off. If you reconfigure the backend for any
reason (e.g. to add a region), repeat every flag you want to keep.

## Quick Start

```bash
# 1. Authenticate (one time)
gcloud auth application-default login

# 2. Configure the backend
aerolab config backend -t gcp -r us-central1 -o your-project-id

# 3. Create a 2-node cluster
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*' \
  --instance e2-standard-4 \
  --gcp.disk type=pd-ssd,size=20 \
  --gcp.expire=8h

# 4. Wait for it to come up, then use it
aerolab aerospike is-stable -w
aerolab attach aql -n asd -- -c "show namespaces"

# 5. Tear it down when done
aerolab cluster destroy -n asd --force
```

`--instance e2-standard-4` picks the instance type, `--gcp.disk type=pd-ssd,size=20` sets a
20GB root disk, and `--gcp.expire=8h` auto-destroys the cluster after 8 hours so you don't
forget it.

## Configure the Backend

```bash
aerolab config backend -t gcp -r us-central1 -o your-project-id
```

Optional flags:

| Flag | Effect |
|------|--------|
| `--inventory-cache` | Cache resource state locally — only if you're the sole user of the project. |
| `--gcp.auto-enable-services` | Enable missing GCP APIs automatically, without prompting (required for CI/non-interactive use). |
| `--gcp.no-public-ip`, `--gcp.use-iap` | See [Gotchas](#gotchas) above. |
| `--skip-pricing` | Skip cost/pricing lookups; needed under Workload Identity Federation, where the billing API rejects federated tokens. |
| `-r` (comma-separated) | Enable multiple regions, e.g. `us-central1,us-east1,us-west1`. |

Verify and check access:

```bash
aerolab config backend
aerolab config backend -t gcp --check-access
```

## GCP-Specific Configuration

Firewall rules are managed with `config gcp`:

```bash
aerolab config gcp list-firewall-rules
aerolab config gcp create-firewall-rules -n aerolab-fw -p 3000-3005
aerolab config gcp lock-firewall-rules -n aerolab-fw
aerolab config gcp delete-firewall-rules -n aerolab-fw
```

## Creating Clusters

Beyond the Quick Start example, common `cluster create` additions:

| Need | Flag(s) |
|------|---------|
| Specific zone | `--zone us-central1-a` |
| Multiple disks | repeat `--gcp.disk type=pd-ssd,size=100,count=3` |
| Different disk type | `--gcp.disk type=hyperdisk-balanced,size=100,iops=3060,throughput=155` |
| Custom firewall rule | `--firewall aerolab-fw` |
| Public IP (overrides backend default) | `--gcp.public-ip` |
| Spot instances (cheaper) | `--gcp.spot` |
| Labels / tags | repeat `--label Key=Value` / `--tag name` |
| Volume mount | `--gcp.vol-create --gcp.vol-mount myvolume:/mnt/data --gcp.vol-size 100` |

## Resource Expiry Automation

Deploys a Cloud Function that auto-destroys expired resources (see
[GCP Services](gcp-services.md) for the APIs it enables):

```bash
aerolab config gcp expiry-install
aerolab config gcp expiry-list
aerolab config gcp expiry-run-frequency -f 20   # minutes
aerolab config gcp expiry-remove
```

## Next: Lifecycle, Attach, Files, Cleanup

Starting/stopping, controlling the Aerospike service, `attach` (shell/aql/asinfo/asadm),
file upload/download, and cleanup are identical across every backend — see
[Common Operations](common-operations.md). One GCP-specific note: stopping instances
doesn't delete their persistent disks, so you're still billed until you `destroy` or let
expiry run.

Cluster-level extras: `aerolab cluster add public-ip -n asd` and
`aerolab cluster add firewall -n asd -f aerolab-fw` add these after the fact.

## Troubleshooting

### Authentication Issues

1. **Application Default Credentials not found** — run
   `gcloud auth application-default login`. If you see "could not authenticate using
   application credentials", follow
   https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment.
2. **Check status:**
   ```bash
   gcloud auth list
   gcloud config get-value project
   ```
3. **Re-authenticate** if credentials expired: `gcloud auth application-default login`.
4. **Check permissions** — your account needs Compute Engine permissions in the project.

### Project / Region Issues

```bash
aerolab config backend               # verify project/region
gcloud compute zones list            # available zones
```

### Instance Type Availability

```bash
aerolab inventory instance-types
```

### Quota Issues

Check quotas in the GCP Console, request increases, or use smaller instances/fewer nodes.

## Next Steps

- [Common Operations](common-operations.md) - lifecycle, attach, files, cleanup
- [Cluster management commands](../commands/cluster.md)
- [GCP-specific volume management](../commands/volumes.md)
- [GCP Services (APIs) Required by AeroLab](gcp-services.md)
