# Client Management Commands

Client commands enable you to create and manage client machines for various purposes including monitoring, development, testing, and administration.

## Commands Overview

- `client create` - Create new client machines (none, base, tools, ams, vscode, graph, eksctl)
- `client grow` - Add machines to existing client groups
- `client configure` - Configure client machines (ams, firewall, tools, expiry)
- `client template` - Manage client template images (pre-built images for faster client creation)
- `client list` - List all client machine groups
- `client start` - Start client machines
- `client stop` - Stop client machines
- `client destroy` - Destroy client machines
- `client attach` - Attach to a client machine (shorthand for `attach client`)
- `client share` - Share client access via SSH public key
- `client update-hosts-file` - Update the hosts file on client machines

## Client Types

### None
Vanilla OS image with no modifications. Useful for custom setups.

```bash
aerolab client create none -n myclient --os ubuntu --version 24.04
```

### Base
Simple base image with basic tools installed.

```bash
aerolab client create base -n myclient --os ubuntu --version 24.04
```

### Tools
Aerospike tools (asbench, asadm, asinfo, asloglatency) pre-installed.

```bash
aerolab client create tools -n tools --os ubuntu --version 24.04
```

### AMS
Aerospike Monitoring Stack with Prometheus, Grafana, and Loki.

```bash
aerolab client create ams -n ams --os ubuntu --version 24.04 \
  -s mycluster -S graph-client
```

**AMS Options:**
- `--grafana-version` - Grafana version (default: `latest`)
- `--prometheus-version` - Prometheus version (default: `latest`)
- `-s, --clusters` - Clusters to monitor (comma-separated)
- `-S, --clients` - Graph clients to monitor (comma-separated)
- `--dashboards` - Custom dashboards YAML file
- `--debug-dashboards` - Enable debug output for dashboard installation

### VSCode
VSCode Server for browser-based development.

```bash
aerolab client create vscode -n ide --os ubuntu --version 24.04
```

### Graph
Graph database client for Aerospike Graph.

```bash
aerolab client create graph -n graph --os ubuntu --version 24.04 \
  -C mycluster
```

`-C, --cluster-name` seeds the graph service from an existing Aerospike
cluster (default: `mydc`); use `--seed` instead to point at a raw `IP:PORT`.

### EksCtl
Client machine with eksctl pre-configured for Kubernetes Aerospike deployments.

```bash
aerolab client create eksctl -n k8s-admin --os ubuntu --version 24.04
```

---

## Client Create

Create new client machines.

### Basic Usage

```bash
aerolab client create <type> -n <name> --os <distro> --version <version> [options]
```

### Common Options

| Option | Description | Default |
|--------|-------------|---------|
| `-n, --group-name` | Client group name | `client` |
| `-c, --count` | Number of client machines | `1` |
| `--os` | OS distribution (ubuntu, centos, rocky, debian, amazon) | `ubuntu` |
| `--version` | OS version (e.g., `24.04`, `22.04`) | `24.04` |
| `--type-override` | Override auto-detected client type | (empty) |
| `--threads` | Number of parallel threads | `10` |
| `-t, --tag` | Tags to add, format `k=v` (can be specified multiple times) | |

**Note:** Backend-specific options are grouped under a namespace, so each flag
is prefixed with `aws.`, `gcp.`, or `docker.` (e.g. `--aws.instance`, not
`--instance-type`) — the same pattern used by [`instances create`](instances.md#instances-create).

### Docker Backend Options

```bash
aerolab client create tools -n tools --os ubuntu --version 24.04 \
  -c 2 --docker.expose 9100:9100
```

| Option | Description |
|--------|-------------|
| `--docker.expose` | Expose ports (format: `[+]{hostPort}:{containerPort}`; `+` maps to next available port) |
| `--docker.network` | Docker network name to attach to (default: `default`) |

### AWS Backend Options

```bash
aerolab client create tools -n tools --os ubuntu --version 24.04 \
  --aws.instance t3a.medium --aws.disk type=gp3,size=20 --aws.expire=4h
```

| Option | Description |
|--------|-------------|
| `--aws.instance` | Instance type (e.g., `t3a.medium`) |
| `--aws.disk` | Disk spec: `type={gp3\|gp2\|io2\|io1},size={GB}[,count=N]` |
| `--aws.expire` | Expiry time (e.g., `4h`, `30m`) |
| `--aws.placement` | Subnet ID, availability zone, or region |
| `--aws.no-public-ip` | Disable public IP assignment |
| `--aws.firewall` | Extra security group names |

### GCP Backend Options

```bash
aerolab client create tools -n tools --os ubuntu --version 24.04 \
  --gcp.instance e2-medium --gcp.disk type=pd-ssd,size=20
```

| Option | Description |
|--------|-------------|
| `--gcp.instance` | Instance type (e.g., `e2-medium`) |
| `--gcp.zone` | Zone name (e.g., `us-central1-a`) |
| `--gcp.disk` | Disk spec: `type=pd-ssd[,size={GB}][,count=N]` |
| `--gcp.expire` | Expiry time |
| `--gcp.no-public-ip` | Disable public IP assignment |
| `--gcp.firewall` | Firewall rule names |

---

## Client Grow

Add more machines to an existing client group.

### Basic Usage

```bash
aerolab client grow <type> -n <name> -c <count>
```

### Examples

```bash
# Add 2 more tools clients
aerolab client grow tools -n tools -c 2

# Add 1 more AMS client
aerolab client grow ams -n ams -c 1 -s mycluster
```

---

## Client Configure

Configure or reconfigure client machines.

### Configure AMS

Reconfigure AMS to monitor different clusters or clients.

```bash
aerolab client configure ams -n ams -s cluster1,cluster2 -S graph1
```

**Options:**
- `-n, --group-name` - Client group name (default: `client`)
- `-l, --machines` - Specific machines, comma separated (default: all)
- `-s, --clusters` - Clusters to monitor (comma-separated)
- `-S, --clients` - Graph clients to monitor (comma-separated)

**Example:**
```bash
# Add monitoring for new cluster
aerolab client configure ams -n ams -s prod-us,prod-eu

# Reconfigure specific AMS machines
aerolab client configure ams -n ams -l 1,2 -s mycluster
```

### Configure Firewall

Assign firewall rules to client machines.

```bash
aerolab client configure firewall -n myclient -f firewall-name
```

**Options:**
- `-n, --group-name` - Client group name (default: `client`)
- `-l, --machines` - Specific machines (default: all)
- `-f, --firewall` - Firewall name to assign (required)

**Example:**
```bash
# Assign firewall to all clients
aerolab client configure firewall -n tools -f allow-outbound

# Assign to specific machines
aerolab client configure firewall -n tools -l 1,2,3 -f restricted
```

### Configure Tools

Configure tools clients to send logs to AMS (Loki).

```bash
aerolab client configure tools -n tools -m ams
```

**Options:**
- `-n, --group-name` - Client group name (default: `client`)
- `-l, --machines` - Specific machines (default: all)
- `-m, --ams` - AMS client machine name (default: `ams`)
- `-t, --threads` - Number of parallel threads (default: `10`)

**What it does:**
- Installs Promtail (log aggregator)
- Configures Promtail to scrape asbench logs
- Sends logs to Loki on AMS client
- Creates systemd service for Promtail
- Enables autostart on boot

**Example:**
```bash
# Configure all tools clients
aerolab client configure tools -n tools -m ams

# Configure specific machines
aerolab client configure tools -n tools -l 1,2 -m my-ams
```

### Configure Expiry

Change (or remove) the expiry time of client machines (AWS/GCP only).

```bash
aerolab client configure expiry -n tools -e 24h
```

**Options:**
- `-n, --group-name` - Client group name (default: `client`)
- `-l, --machines` - Specific machines (default: all)
- `-e, --expiry` - Expiry duration from now, e.g. `1D12h`, `2W`, `1Y6M` (default: `30h`; use `0` to remove expiry)

---

## Client List

List all client machine groups.

### Basic Usage

```bash
aerolab client list
```

### Example Output

```
Client Groups:
  Name: ams
    Type: ams
    Count: 1
    State: running
    IPs: 10.0.1.100
  
  Name: tools
    Type: tools
    Count: 3
    State: running
    IPs: 10.0.1.101, 10.0.1.102, 10.0.1.103
```

---

## Client Start/Stop

Start or stop client machines.

### Basic Usage

```bash
# Start clients
aerolab client start -n tools

# Stop clients
aerolab client stop -n tools
```

**Options:**
- `-n, --group-name` - Client group name (default: `client`)
- `-l, --machines` - Specific machines (default: all)

---

## Client Destroy

Destroy client machines.

### Basic Usage

```bash
aerolab client destroy -n tools
```

**Options:**
- `-n, --group-name` - Client group name (default: `client`)
- `-l, --machines` - Specific machines (default: all)
- `-f, --force` - Force destruction without confirmation

**Examples:**
```bash
# Destroy entire client group
aerolab client destroy -n tools -f

# Destroy specific machines
aerolab client destroy -n tools -l 1,2 -f

# Destroy multiple groups
aerolab client destroy -n tools,ams,vscode -f
```

---

## Client Share

Share client access with other users via SSH public key.

### Basic Usage

```bash
aerolab client share -n tools -f ~/.ssh/id_rsa.pub
```

**Options:**
- `-n, --name` - Client name (default: `client`)
- `-f, --pubkey` - Path to the SSH public key to import
- `-p, --parallel-threads` - Number of parallel threads (default: `10`)

**Note:** unlike `configure`/`start`/`stop`/`destroy`, `client share` has no
`-l, --machines` filter — it applies to every machine in the named group(s).

**Example:**
```bash
# Share access to all machines in the ams group
aerolab client share -n ams -f ~/.ssh/team_key.pub

# Share with multiple groups
aerolab client share -n tools,ams -f ~/.ssh/developer.pub
```

---

## Client Update Hosts File

Update the /etc/hosts file on client machines with cluster information.

```bash
aerolab client update-hosts-file
```

**Options:** `-o, --on` (update hosts file only on these clusters), `-w, --with`
(include only instances from these clusters in the generated file); both
default to all clusters.

---

## Complete Examples

### AMS Monitoring Setup

```bash
# 1. Create Aerospike cluster
aerolab cluster create -n prod -c 3 -d ubuntu -i 24.04 -v '8.*'

# 2. Add exporter to cluster
aerolab cluster add exporter -n prod

# 3. Create AMS client
aerolab client create ams -n ams --os ubuntu --version 24.04 -s prod

# 4. Create tools client for benchmarking
aerolab client create tools -n tools --os ubuntu --version 24.04

# 5. Configure tools to send logs to AMS
aerolab client configure tools -n tools -m ams

# 6. Access Grafana (check client list for IP)
aerolab client list
# Open browser to http://<ams-ip>:3000
# Username: admin, Password: admin
```

### Development Environment

```bash
# Create VSCode IDE client
aerolab client create vscode -n ide --os ubuntu --version 24.04

# Create tools client
aerolab client create tools -n dev-tools --os ubuntu --version 24.04

# Create graph client
aerolab client create graph -n graph-dev --os ubuntu --version 24.04 -C mycluster

# Share access with team
aerolab client share -n ide,dev-tools,graph-dev -f ~/.ssh/team.pub
```

### Multi-Region XDR with Monitoring

```bash
# Create clusters
aerolab xdr create-clusters -n us-east -N eu-west,ap-south \
  -c 3 -C 3 -d ubuntu -i 24.04 -v '8.*'

# Add exporters
aerolab cluster add exporter -n us-east,eu-west,ap-south

# Create AMS for monitoring all regions
aerolab client create ams -n global-ams --os ubuntu --version 24.04 \
  -s us-east,eu-west,ap-south

# Access monitoring
aerolab client list
```

### EKS/Kubernetes Setup

```bash
# Create eksctl client
aerolab client create eksctl -n k8s-admin --os ubuntu --version 24.04

# Attach to client
aerolab client attach -n k8s-admin -l 1

# Inside client: deploy Aerospike Kubernetes Operator
eksctl create cluster --name=aerospike-k8s --region=us-east-1
kubectl apply -f https://operatorhub.io/install/aerospike-kubernetes-operator.yaml
```

## Best Practices

1. **Naming**: Use descriptive names for client groups (e.g., `prod-ams`, `dev-tools`)
2. **Monitoring**: Always create AMS client for production clusters
3. **Tools Integration**: Configure tools clients to send logs to AMS
4. **Access Control**: Use `client share` to manage team access
5. **Resource Cleanup**: Destroy unused clients to save resources
6. **Expiry**: Set appropriate expiry times for cloud-based clients

## Troubleshooting

### AMS Not Showing Metrics

```bash
# Check if exporter is installed on cluster
aerolab cluster add exporter -n mycluster

# Verify Prometheus targets
aerolab attach shell -n ams -l 1
curl http://localhost:9090/api/v1/targets

# Reconfigure AMS
aerolab client configure ams -n ams -s mycluster
```

### Tools Client Logs Not in Loki

```bash
# Check Promtail status
aerolab attach shell -n tools -l 1
systemctl status promtail

# Reconfigure tools
aerolab client configure tools -n tools -m ams

# Check Promtail logs
journalctl -u promtail -f
```

### Cannot Access Grafana

```bash
# Check if port is exposed (Docker)
aerolab client list

# Verify Grafana is running
aerolab attach shell -n ams -l 1
systemctl status grafana-server

# Restart Grafana
systemctl restart grafana-server
```

## See Also

- [Cluster Management](cluster.md) - Managing Aerospike clusters
- [Attach Commands](attach.md) - Accessing client machines
- [Files Commands](files.md) - File operations
- [XDR Commands](xdr.md) - XDR configuration
- [TLS Commands](tls.md) - TLS certificates

