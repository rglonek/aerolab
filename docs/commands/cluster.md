# Cluster Management Commands

Cluster management commands allow you to create, manage, and operate Aerospike clusters across all supported backends.

## Commands Overview

- `cluster create` - Create a new cluster
- `cluster grow` - Add nodes to an existing cluster
- `cluster apply` - Automatically grow/shrink cluster to match desired size
- `cluster list` - List all clusters
- `cluster start` - Start cluster nodes
- `cluster stop` - Stop cluster nodes
- `cluster destroy` - Destroy a cluster
- `cluster add` - Add features to clusters (exporter, aerolab, firewall, public IP)
- `cluster partition` - Manage disk partitions for clusters
- `cluster attach` - Attach to cluster nodes (shorthand for `attach shell`)
- `cluster share` - Share cluster access via SSH public key
- `cluster update-hosts-file` - Update the hosts file on cluster nodes

## Cluster Create

Create a new Aerospike cluster.

### Basic Usage

```bash
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*'
```

### Common Options

| Option | Description | Default |
|--------|-------------|---------|
| `-n, --name` | Cluster name | `asd` |
| `-c, --count` | Number of nodes | `1` |
| `-d, --distro` | Distribution (ubuntu, centos, rocky, debian, amazon) | Required |
| `-i, --distro-version` | Distribution version (e.g., 24.04, 22.04) | Required |
| `-v, --aerospike-version` | Aerospike version (supports wildcards like `'8.*'`) | Required |
| `-s, --start` | Auto-start Aerospike after creation (y/n) | `y` |
| `-f, --feature-file` | Features file or directory containing feature files | |
| `-o, --custom-conf` | Custom aerospike config file path | |
| `-z, --tools-conf` | Custom astools config file path | |
| `-m, --heartbeat-mode` | Heartbeat mode (mcast/mesh/default) | `mesh` |
| `-p, --threads` | Number of parallel threads | `10` |

### Docker Backend

```bash
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*'
```

**Docker Options:**
- `--docker.network` - Name of a non-default Docker network to attach the nodes to. Manage networks with `aerolab config docker list-networks` / `prune-networks`.

**Example (custom network):**
```bash
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*' --docker.network my-aerolab-net
```

### AWS Backend

```bash
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*' \
  --aws.instance t3a.xlarge \
  --aws.disk type=gp3,size=20 \
  --aws.disk type=gp3,size=100,count=3 \
  --aws.expire=8h
```

**AWS Options:**
- `--aws.instance` - Instance type (e.g., `t3a.xlarge`)
- `--aws.disk` - Disk specification: `type={gp3|gp2|io2|io1},size={GB}[,iops={cnt}][,throughput={mb/s}][,count=5]`
- `--aws.expire` - Expiry time (e.g., `8h`, `30m`, `2h30m`)
- `--aws.placement` - Subnet ID or availability zone
- `--aws.public-ip` - Enable public IP access address
- `--aws.spot` - Use spot instances
- `--aws.firewall` - Security group names (can specify multiple)
- `-t, --tag` - Custom tags (format: `key=value`, can specify multiple)
- `--aws.efs-create` - Create EFS volume
- `--aws.efs-mount` - Mount EFS volume (format: `NAME:MountPath`)

### GCP Backend

```bash
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*' \
  --gcp.instance e2-standard-4 \
  --gcp.disk type=pd-ssd,size=20 \
  --gcp.disk type=pd-ssd,size=100,count=3 \
  --gcp.expire=8h
```

**GCP Options:**
- `--gcp.instance` - Instance type (e.g., `e2-standard-4`)
- `--gcp.zone` - Zone name (e.g., `us-central1-a`)
- `--gcp.disk` - Disk specification: `type={pd-*|hyperdisk-*|local-ssd}[,size={GB}][,iops={cnt}][,throughput={mb/s}][,count=5]`
- `--gcp.expire` - Expiry time
- `--gcp.public-ip` - Enable public IP
- `--gcp.spot` - Use spot instances
- `--gcp.firewall` - Firewall rule names (can specify multiple)
- `-t, --tag` - Custom tags/labels (format: `key=value`, can specify multiple)


### Examples

**Create a 3-node cluster with custom name:**
```bash
aerolab cluster create -n production -c 3 -d ubuntu -i 24.04 -v '8.*' \
  --aws.instance t3a.xlarge \
  --aws.disk type=gp3,size=20 \
  --aws.expire=24h
```

**Create cluster without auto-start:**
```bash
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*' --start n
```

**Create cluster with features file:**
```bash
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*' \
  -f /path/to/features.conf
```

**Create cluster with custom configuration:**
```bash
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*' \
  -o /path/to/aerospike.conf
```

## Cluster Grow

Add nodes to an existing cluster.

### Basic Usage

```bash
aerolab cluster grow -c 2
```

This adds 2 nodes to the default cluster (`asd`).

### Options

All options from `cluster create` are available, but the cluster must already exist.

### Examples

**Add 2 nodes to a specific cluster:**
```bash
aerolab cluster grow -n asd -c 2 -d ubuntu -i 24.04 -v '8.*' \
  --aws.instance t3a.xlarge \
  --aws.disk type=gp3,size=20 \
  --aws.expire=8h
```

**Note**: When growing, expiry defaults match the existing cluster.

## Cluster Apply

Automatically create, grow, or shrink a cluster to match the desired size.

### Basic Usage

```bash
aerolab cluster apply -c 5
```

This will:
- Create the cluster if it doesn't exist
- Grow the cluster if it has fewer nodes
- Shrink the cluster if it has more nodes (requires `--force`)

### Options

| Option | Description |
|--------|-------------|
| `--force` | Required when shrinking cluster |

### Examples

**Grow cluster to 5 nodes:**
```bash
aerolab cluster apply -c 5 -d ubuntu -i 24.04 -v '8.*' \
  --aws.instance t3a.xlarge \
  --aws.disk type=gp3,size=20 \
  --aws.expire=8h
```

**Shrink cluster to 3 nodes:**
```bash
aerolab cluster apply -c 3 -d ubuntu -i 24.04 -v '8.*' --force \
  --aws.instance t3a.xlarge \
  --aws.disk type=gp3,size=20 \
  --aws.expire=8h
```

## Cluster List

List all clusters.

### Basic Usage

```bash
aerolab cluster list
```

### Output Formats

**Table format (default):**
```bash
aerolab cluster list
```

**TSV format:**
```bash
aerolab cluster list -o tsv
```

**JSON format:**
```bash
aerolab cluster list -o json
```

## Cluster Start

Start cluster nodes.

### Basic Usage

```bash
aerolab cluster start
```

Start all nodes in the default cluster.

### Options

| Option | Description |
|--------|-------------|
| `-n, --name` | Cluster name (comma-separated for multiple) |
| `-l, --nodes` | Node list (comma-separated or ranges like `1-3`) |

### Examples

**Start all nodes in default cluster:**
```bash
aerolab cluster start
```

**Start specific nodes:**
```bash
aerolab cluster start -n asd -l 1-2
```

**Start multiple clusters:**
```bash
aerolab cluster start -n asd,otherdc
```

## Cluster Stop

Stop cluster nodes.

### Basic Usage

```bash
aerolab cluster stop
```

### Options

Same as `cluster start`.

### Examples

**Stop all nodes:**
```bash
aerolab cluster stop
```

**Stop specific nodes:**
```bash
aerolab cluster stop -n asd -l 1-2
```

## Cluster Destroy

Destroy a cluster and all its resources.

### Basic Usage

```bash
aerolab cluster destroy -n asd --force
```

### Options

| Option | Description |
|--------|-------------|
| `-n, --name` | Cluster name (comma-separated for multiple) |
| `--force` | Force destruction without confirmation |

### Examples

**Destroy a cluster:**
```bash
aerolab cluster destroy -n asd --force
```

**Destroy multiple clusters:**
```bash
aerolab cluster destroy -n asd,otherdc --force
```

## Cluster Add

Add features to clusters.

### Available Subcommands

- `cluster add exporter` - Add Prometheus exporter
- `cluster add aerolab` - Add Aerolab tools
- `cluster add firewall` - Add firewall rules (AWS/GCP)
- `cluster add public-ip` - Add public IP access (AWS/GCP)
- `cluster add expiry` - Add expiry automation (AWS/GCP)

### Add Exporter

Install Prometheus exporter on cluster nodes:

```bash
aerolab cluster add exporter
```

### Add Aerolab

Install Aerolab tools on cluster nodes:

```bash
aerolab cluster add aerolab
```

### Add Firewall

Add firewall rules to cluster (AWS/GCP only):

```bash
aerolab cluster add firewall -n asd -f firewall-name
```

### Add Public IP

Add public IP access to cluster (AWS/GCP only):

```bash
aerolab cluster add public-ip -n asd
```

## Cluster Partition

Manage disk partitions for clusters (AWS/GCP only).

### Subcommands

- `cluster partition create` - Create partitions
- `cluster partition list` - List partitions
- `cluster partition conf` - Configure partitions
- `cluster partition mkfs` - Format partitions

### Create Partitions

Create disk partitions:

```bash
aerolab cluster partition create -p 25,25,25,25
```

This creates 4 partitions, each 25% of the total disk space (percentages must sum to 100 or less).

**Options:**
- `-p, --partitions` - Partition sizes as a percentage of total disk space (comma-separated); omit to remove all partitions
- `-n, --name` - Cluster name
- `-l, --nodes` - Node list

### List Partitions

List partitions on nodes:

```bash
aerolab cluster partition list -n asd
```

### Configure Partitions

Configure partitions for Aerospike namespaces:

```bash
aerolab cluster partition conf --namespace=test --configure=device --filter-partitions=3-6
```

**Configuration Types:**
- `device` - Configure as device
- `pi-flash` - Configure as persistent index (flash)
- `si-flash` - Configure as storage index (flash)

**Options:**
- `--namespace` - Namespace name
- `--configure` - Configuration type
- `--filter-partitions` - Partition numbers (comma-separated or ranges)

### Format Partitions

Format partitions:

```bash
aerolab cluster partition mkfs --filter-partitions=1,2
```

**Options:**
- `--filter-partitions` - Partition numbers to format
- `-n, --name` - Cluster name
- `-l, --nodes` - Node list

## Cluster Attach

Attach to cluster nodes (shorthand for `attach shell`).

### Basic Usage

```bash
aerolab cluster attach -n asd -l 1
```

### Options

| Option | Description |
|--------|-------------|
| `-n, --name` | Cluster name |
| `-l, --nodes` | Node list (`all` for all nodes) |
| `-p, --parallel` | Execute in parallel on all nodes |

### Examples

**Attach to a node:**
```bash
aerolab cluster attach -n asd -l 1
```

**Run command on all nodes:**
```bash
aerolab cluster attach -n asd -l all -- ls /tmp
```

**Run command in parallel:**
```bash
aerolab cluster attach -n asd -l all --parallel -- ls /tmp
```

## Cluster Share

Share cluster access via SSH public key (AWS/GCP only).

### Basic Usage

```bash
aerolab cluster share -n asd -f /path/to/public-key.pub
```

This imports the SSH public key to allow access to cluster nodes.

### Options

| Option | Description |
|--------|-------------|
| `-n, --name` | Cluster name |
| `-l, --nodes` | Node list, comma separated (default: all) |
| `-f, --pubkey` | Path to SSH public key file |
| `-p, --threads` | Number of parallel threads (default: `10`) |

## Common Workflows

### Create and Configure a Production Cluster

```bash
# 1. Create cluster
aerolab cluster create -n production -c 5 -d ubuntu -i 24.04 -v '8.*' \
  --aws.instance t3a.xlarge \
  --aws.disk type=gp3,size=20 \
  --aws.disk type=gp3,size=100,count=3 \
  --aws.expire=24h

# 2. Create partitions
aerolab cluster partition create -n production -p 16,16,16,16,16,16

# 3. Format partitions
aerolab cluster partition mkfs -n production --filter-partitions=1,2

# 4. Configure partitions
aerolab cluster partition conf -n production --namespace=test \
  --configure=pi-flash --filter-partitions=1
aerolab cluster partition conf -n production --namespace=test \
  --configure=si-flash --filter-partitions=2

# 5. Start cluster
aerolab cluster start -n production

# 6. Start Aerospike
aerolab aerospike start -n production

# 7. Wait for stability
aerolab aerospike is-stable -n production -w
```

### Scale Cluster Up and Down

```bash
# Scale up to 10 nodes
aerolab cluster apply -n asd -c 10 -d ubuntu -i 24.04 -v '8.*' \
  --aws.instance t3a.xlarge \
  --aws.disk type=gp3,size=20 \
  --aws.expire=8h

# Scale down to 5 nodes
aerolab cluster apply -n asd -c 5 -d ubuntu -i 24.04 -v '8.*' --force \
  --aws.instance t3a.xlarge \
  --aws.disk type=gp3,size=20 \
  --aws.expire=8h
```

### Add Features to Existing Cluster

```bash
# Add exporter
aerolab cluster add exporter -n asd

# Add public IP
aerolab cluster add public-ip -n asd

# Add firewall rules
aerolab cluster add firewall -n asd -f aerolab-sg
```

## Tips

1. **Cluster Names**: Use descriptive cluster names to manage multiple clusters
2. **Node Filtering**: Use `-l` to filter operations to specific nodes
3. **Expiry**: Set expiry times on resources to auto-cleanup
4. **Parallel Operations**: Use `--parallel` for faster operations on multiple nodes
5. **Partition Management**: Configure partitions before starting Aerospike for best performance

