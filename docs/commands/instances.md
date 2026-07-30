# Instance Management Commands

Instance management commands provide low-level control over compute instances without Aerospike-specific configurations. These are the building blocks used by cluster commands.

## Commands Overview

- `instances create` - Create a new instance cluster
- `instances grow` - Grow an existing instance cluster
- `instances apply` - Automatically create, grow, or shrink to match desired cluster size
- `instances list` - List all instances
- `instances attach` - Attach to an instance or cluster
- `instances start` - Start instances
- `instances stop` - Stop instances
- `instances restart` - Restart instances
- `instances update-hosts-file` - Update the hosts file on instances
- `instances add-tags` - Add tags to instances
- `instances remove-tags` - Remove tags from instances
- `instances assign-firewalls` - Assign firewalls/security groups to instances
- `instances remove-firewalls` - Remove firewalls/security groups from instances
- `instances change-expiry` - Change the expiry time of instances
- `instances destroy` - Destroy instances

## When to Use Instances vs Clusters

- **Use `cluster` commands** for Aerospike clusters (includes Aerospike installation and configuration)
- **Use `instances` commands** for:
  - Creating custom environments
  - Non-Aerospike workloads
  - Fine-grained control over infrastructure
  - Building custom automation

## Instances Create

Create raw compute instances without Aerospike installation.

### Basic Usage

```bash
aerolab instances create -n myinstances -c 2 --os ubuntu --version 24.04
```

### Common Options

| Option | Description | Default |
|--------|-------------|---------|
| `-n, --cluster-name` | Instance cluster name | `asd` |
| `-c, --count` | Number of instances | `1` |
| `-N, --name` | Name of a single instance (only when `--count` is 1) | |
| `--os` | OS distribution (ubuntu, centos, rocky, debian, amazon) | `ubuntu` |
| `--version` | OS version (e.g., `24.04`, `22.04`) | `24.04` |
| `--arch` | Architecture override (`amd64`, `arm64`) | |
| `-p, --threads` | Number of parallel SSH threads | `10` |
| `-t, --tag` | Tags to add, format `k=v` (can be specified multiple times) | |

**Note:** Backend-specific options below are grouped under a namespace, so each
flag is prefixed with `aws.`, `gcp.`, or `docker.` (e.g. `--aws.instance`, not
`--instance-type`). The same pattern is used by `cluster create` and
`client create`.

### Backend-Specific Options

**AWS (`--aws.*`):**
- `--aws.instance` - Instance type
- `--aws.disk` - Disk specification: `type={gp3|gp2|io2|io1},size={GB}[,iops={cnt}][,throughput={mb/s}][,count=5][,encrypted=true|false]`
- `--aws.expire` - Expiry time (default: `30h`)
- `--aws.placement` - Region name, VPC-ID, or subnet-ID
- `--aws.firewall` - Extra security group names (can be specified multiple times)
- `--aws.no-public-ip` - Disable public IP assignment
- `--aws.spot` - Create a spot instance

**GCP (`--gcp.*`):**
- `--gcp.instance` - Instance type
- `--gcp.zone` - Zone name
- `--gcp.disk` - Disk specification: `type={pd-*,hyperdisk-*,local-ssd}[,size={GB}][,iops={cnt}][,throughput={mb/s}][,count=5]`
- `--gcp.expire` - Expiry time (default: `30h`)
- `--gcp.firewall` - Extra firewall rule names (can be specified multiple times)
- `--gcp.no-public-ip` - Disable public IP assignment
- `--gcp.spot` - Create a spot instance

**Docker (`--docker.*`):**
- `--docker.network` - Name of the Docker network to attach the instances to (default: `default`)
- `--docker.expose` - Port exposure, format `[+]{hostPort}:{containerPort}`

### Examples

**AWS:**
```bash
aerolab instances create -n myinstances -c 2 --os ubuntu --version 24.04 \
  --aws.instance t3a.xlarge --aws.disk type=gp3,size=20 --aws.expire=8h
```

**GCP:**
```bash
aerolab instances create -n myinstances -c 2 --os ubuntu --version 24.04 \
  --gcp.instance e2-standard-4 --gcp.disk type=pd-ssd,size=20 --gcp.expire=8h
```

**Docker:**
```bash
aerolab instances create -n myinstances -c 2 --os ubuntu --version 24.04 \
  --docker.network my-aerolab-net
```

## Instances Grow

Add more instances to an existing instance cluster.

```bash
aerolab instances grow -n myinstances -c 2 --os ubuntu --version 24.04
```

## Instances Apply

Automatically adjust the instance cluster to match a desired size.

```bash
# Grow to 5 instances
aerolab instances apply -n myinstances -c 5

# Shrink to 2 instances (requires --force)
aerolab instances apply -n myinstances -c 2 --force
```

**Note:** `instances apply` does not take `--os`/`--version` — it reuses the
existing cluster's configuration when growing.

This command will:
- Create the cluster if it doesn't exist
- Grow the cluster if current size < desired size
- Shrink the cluster if current size > desired size (with `--force`)
- Do nothing if already at desired size

## Instance Filters

`list`, `attach`, `start`, `stop`, `restart`, `destroy`, `add-tags`,
`remove-tags`, `assign-firewalls`, `remove-firewalls`, and `change-expiry` all
select their target instances through a shared, **namespaced** filter group
(`--filter.*`), not the `-n`/`-l` flags used by `cluster` commands:

| Option | Description |
|--------|-------------|
| `--filter.cluster-name` | Filter by cluster name |
| `--filter.node-no` | Filter by node number(s), e.g. `1,2,3,10-15` |
| `--filter.name` | Filter by instance name |
| `--filter.owner` | Filter by owner |
| `--filter.type` | Filter by instance type (`aerolab.type` tag) |
| `--filter.backend` | Filter by backend type |
| `--filter.tag` | Filter by tag, format `k=v` |

Omitting all filters targets every instance across all clusters.

## Instances List

List all instances across all clusters.

```bash
# List all instances
aerolab instances list

# List specific cluster
aerolab instances list --filter.cluster-name myinstances

# JSON output
aerolab instances list -o json

# TSV output
aerolab instances list -o tsv
```

## Instances Start/Stop/Restart

Control instance power state.

```bash
# Start all instances
aerolab instances start

# Start specific cluster
aerolab instances start --filter.cluster-name myinstances

# Start specific nodes
aerolab instances start --filter.cluster-name myinstances --filter.node-no 1-3

# Stop instances
aerolab instances stop --filter.cluster-name myinstances

# Restart instances
aerolab instances restart --filter.cluster-name myinstances
```

## Instances Attach

Attach to instances and run commands.

```bash
# Attach to all instances
aerolab instances attach -- ls /tmp

# Attach to a specific cluster
aerolab instances attach --filter.cluster-name myinstances -- hostname

# Attach to specific nodes
aerolab instances attach --filter.cluster-name myinstances --filter.node-no 1,3 -- uptime
```

## Instances Update Hosts File

Update the /etc/hosts file on instances with cluster information.

```bash
aerolab instances update-hosts-file
```

**Options:** `-o, --on` (update hosts file only on these clusters), `-w, --with`
(include only instances from these clusters in the generated file); both
default to all clusters.

## Tag Management

Add or remove tags from instances (AWS/GCP only).

```bash
# Add tags
aerolab instances add-tags --filter.cluster-name myinstances --tag env=production --tag team=devops

# Remove tags
aerolab instances remove-tags --filter.cluster-name myinstances --tag team
```

## Firewall Management

Assign or remove firewalls/security groups from instances (AWS/GCP only).

```bash
# Assign firewall
aerolab instances assign-firewalls --filter.cluster-name myinstances --firewall my-custom-fw

# Remove firewall
aerolab instances remove-firewalls --filter.cluster-name myinstances --firewall my-custom-fw
```

## Change Expiry

Change the expiry time of instances (AWS/GCP only).

```bash
# Set new expiry time
aerolab instances change-expiry --filter.cluster-name myinstances --expire-in 24h

# Remove expiry (set to 0)
aerolab instances change-expiry --filter.cluster-name myinstances --expire-in 0
```

## Instances Destroy

Destroy instances.

```bash
# Destroy entire cluster (requires --force)
aerolab instances destroy --filter.cluster-name myinstances --force

# Destroy specific nodes
aerolab instances destroy --filter.cluster-name myinstances --filter.node-no 4-5 --force
```

## Differences from Cluster Commands

| Feature | instances | cluster |
|---------|-----------|---------|
| Aerospike Installation | ❌ No | ✅ Yes |
| Aerospike Configuration | ❌ No | ✅ Yes |
| Auto-start Aerospike | ❌ No | ✅ Yes (optional) |
| Templates | ❌ No | ✅ Yes |
| Feature Files | ❌ No | ✅ Yes |
| Use Case | Raw infrastructure | Aerospike clusters |

## See Also

- [Cluster Management](cluster.md) - High-level cluster management with Aerospike
- [Templates](templates.md) - Manage instance templates
- [Images](images.md) - Manage system images

