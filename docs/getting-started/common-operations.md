# Common Operations (All Backends)

Once a cluster exists, these commands behave the same on Docker, AWS, and GCP. The
backend-specific getting started guides ([Docker](docker.md), [AWS](aws.md), [GCP](gcp.md))
link here instead of repeating this section.

## Starting and Stopping Clusters

```bash
# Start all nodes
aerolab cluster start

# Start specific nodes
aerolab cluster start -n mydc -l 1-2

# Stop all nodes
aerolab cluster stop

# Stop specific nodes
aerolab cluster stop -n mydc -l 1-2
```

**Note (AWS/GCP):** stopping instances doesn't delete them — you're still billed for
attached disks/EBS volumes. Use expiry or `cluster destroy` to fully remove resources.

## Managing the Aerospike Service

```bash
aerolab aerospike start
aerolab aerospike stop
aerolab aerospike restart
aerolab aerospike status

# Wait for the cluster to report stable
aerolab aerospike is-stable -w

# With a timeout (seconds)
aerolab aerospike is-stable -w -o 30
```

## Connecting to Nodes

```bash
# Shell access
aerolab attach shell -n mydc -l 1

# Run a single command
aerolab attach shell -n mydc -l 1 -- ls /tmp

# AQL (Aerospike Query Language)
aerolab attach aql -n mydc -- -c "show namespaces"

# asinfo
aerolab attach asinfo -n mydc -- -v "cluster-stable"

# asadm (Aerospike Admin)
aerolab attach asadm -n mydc -- -e info
```

## File Operations

```bash
# Upload
aerolab files upload -n mydc local-file.txt /tmp/remote-file.txt

# Download
aerolab files download -n mydc /tmp/remote-file.txt ./local-dir/

# Sync a file from node 1 to every other node in the cluster
aerolab files sync -n mydc -l 1 /tmp/file.txt
```

## Configuration Management

```bash
# View the running config
aerolab attach shell -n mydc -- cat /etc/aerospike/aerospike.conf

# Adjust a parameter live
aerolab conf adjust set network.heartbeat.interval 250

# Fix mesh configuration after nodes changed
aerolab conf fix-mesh

# Assign rack IDs
aerolab conf rackid -l 1-2 -i 1
aerolab conf rackid -l 3-4 -i 2
```

## Cleanup

```bash
# Destroy one cluster
aerolab cluster destroy -n mydc --force

# Remove every Aerolab-managed resource in the current backend/project
aerolab inventory delete-project-resources -f

# Same, but only resources past their expiry (AWS/GCP)
aerolab inventory delete-project-resources -f --with-expiry
```

## See Also

- [Cluster Management](../commands/cluster.md)
- [Aerospike Daemon Controls](../commands/aerospike.md)
- [Configuration File Management](../commands/conf.md)
- [File Operations](../commands/files.md)
- [Attach Commands](../commands/attach.md)
