# Getting Started with Docker Backend

Docker is the fastest way to run Aerolab: no cloud account, no credentials, clusters in
seconds. It requires Docker, Docker Desktop, Podman, or Podman Desktop to be installed and
running.

## Prerequisites

Install one of:

- **Docker** - [Install Docker](https://docs.docker.com/get-docker/)
- **Docker Desktop** - [Install Docker Desktop](https://www.docker.com/products/docker-desktop/)
- **Podman** - [Install Podman](https://podman.io/getting-started/installation)
- **Podman Desktop** - [Install Podman Desktop](https://podman.io/getting-started/installation)

```bash
docker --version   # or: podman --version
```

## Gotchas

- **Podman is auto-detected.** Aerolab inspects the daemon it connects to and adapts
  automatically — you don't tell it whether it's talking to Docker or Podman.
- **Architecture mismatches.** By default Aerolab builds/runs images for your host's native
  architecture. If you need the other one (e.g. running arm64 images on an amd64 host) your
  Docker/Podman install needs multiarch (QEMU) support, and you must force it explicitly:
  `aerolab config backend -t docker --docker.arch amd64` (or `arm64`).
- **WSL2**: if you don't pass `--temp-dir`, Aerolab detects WSL2 automatically (via `uname -r`)
  and switches its temp directory to `~/.aerolab.tmp` for you — no action needed.
- **Permission denied talking to the Docker socket** usually means your user isn't in the
  `docker` group — see [Troubleshooting](#permission-issues) below.

## Quick Start

```bash
# 1. Configure the backend
aerolab config backend -t docker

# 2. Create a 2-node cluster
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*'

# 3. Wait for it to come up, then use it
aerolab aerospike is-stable -w
aerolab attach aql -n asd -- -c "show namespaces"

# 4. Tear it down when done
aerolab cluster destroy -n asd --force
```

`-c 2` creates 2 nodes, `-d ubuntu -i 24.04` picks the OS image, `-v '8.*'` installs the
latest Aerospike 8.x, and the cluster is named `asd` unless you pass `-n`.

## Configure the Backend

```bash
aerolab config backend -t docker
```

Optional flags:

| Flag | Effect |
|------|--------|
| `--inventory-cache` | Cache resource state locally for faster operations. Only use this if you're not sharing the Docker host with other users. |
| `--docker.arch amd64\|arm64` | Force a specific architecture (see [Gotchas](#gotchas)). |
| `--docker.registry-region na\|eu\|disabled` | Region for the pre-built template image registry. |
| `-d, --temp-dir <path>` | Custom temp directory (see the WSL2 gotcha above). |

Verify what's configured:

```bash
aerolab config backend
```

## Creating Clusters

```bash
# Custom name
aerolab cluster create -n mycluster -c 2 -d ubuntu -i 24.04 -v '8.*'

# Don't auto-start Aerospike after creation
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*' --start n

# See what Aerospike versions are available
aerolab installer list-versions

# List clusters / all resources
aerolab cluster list
aerolab inventory list
```

## Next: Lifecycle, Attach, Files, Cleanup

Starting/stopping, controlling the Aerospike service, `attach` (shell/aql/asinfo/asadm),
file upload/download, and cleanup are identical across every backend — see
[Common Operations](common-operations.md).

## Troubleshooting

### Docker Not Running

```bash
docker ps
```

### Permission Issues

```bash
sudo usermod -aG docker $USER
```

Then log out and back in.

### Network Issues

```bash
docker network ls
aerolab config docker list-networks
```

Clean up unused Docker networks:

```bash
aerolab config docker prune-networks
```

### Clean Up Failed Resources

```bash
aerolab inventory delete-project-resources -f
```

## Next Steps

- [Common Operations](common-operations.md) - lifecycle, attach, files, cleanup
- [Cluster management commands](../commands/cluster.md)
- [Aerospike daemon controls](../commands/aerospike.md)
- [Configuration management](../commands/conf.md)
