# Getting Started with AWS Backend

The AWS backend creates and manages Aerospike clusters on EC2. Use it for production-like
testing and performance benchmarks.

## Prerequisites

1. An active AWS account
2. The [AWS CLI](https://aws.amazon.com/cli/) installed
3. Credentials configured — pick one:

```bash
# Option 1: interactive (recommended)
aws configure

# Option 2: environment variables
export AWS_ACCESS_KEY_ID=your-access-key-id
export AWS_SECRET_ACCESS_KEY=your-secret-access-key
export AWS_DEFAULT_REGION=us-east-1

# Option 3: a named profile (if you have multiple accounts)
aws configure --profile myprofile
# then tell aerolab to use it:
aerolab config backend -t aws -r us-east-1 -P myprofile
```

Credentials live in `~/.aws/credentials`, config in `~/.aws/config`.

**Required IAM permissions:** EC2 (instances, images, volumes, security groups, VPCs), plus
`iam:GetRole`/`iam:PassRole` if you use instance profiles, and Route53/EKS if you use DNS or
an EKS cluster name. A minimal policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["ec2:*", "iam:GetRole", "iam:PassRole"],
      "Resource": "*"
    }
  ]
}
```

## Gotchas

### Optional: Disable Public IPs

`--aws.no-public-ip` stops Aerolab from requesting a public IP for new instances:

```bash
aerolab config backend -t aws -r us-east-1 --aws.no-public-ip
```

**There is no AWS equivalent of GCP's IAP tunnel.** This flag only removes the public IP
from the EC2 launch request — it does not set up SSM Session Manager, a VPN, or any other
path to the instance. You must already have private network access (VPN, VPC peering,
Direct Connect, or a bastion/Session Manager you configure yourself); Aerolab does not check
reachability and has no fallback if you don't. Combining `--aws.no-public-ip` with
`-L`/`--public-ip`/`--gcp.public-ip` on `cluster create` is rejected with an error.

### Flags don't persist across `config backend` runs

`--aws.no-public-ip` and `--skip-pricing` (and their GCP equivalents) are **not** merged with
your last saved config — omitting a previously-set flag silently turns it back off. If
you've configured the backend with `--aws.no-public-ip` and later run `config backend` again
to change something else (e.g. add `--inventory-cache`), you must repeat
`--aws.no-public-ip` too, or it reverts to requesting public IPs.

## Quick Start

```bash
# 1. Configure the backend
aerolab config backend -t aws -r us-east-1

# 2. Create a 2-node cluster
aerolab cluster create -c 2 -d ubuntu -i 24.04 -v '8.*' \
  -I t3a.xlarge \
  --aws.disk type=gp3,size=20 \
  --aws.expire=8h

# 3. Wait for it to come up, then use it
aerolab aerospike is-stable -w
aerolab attach aql -n asd -- -c "show namespaces"

# 4. Tear it down when done
aerolab cluster destroy -n asd --force
```

`-I t3a.xlarge` picks the instance type, `--aws.disk type=gp3,size=20` sets a 20GB root
disk, and `--aws.expire=8h` auto-destroys the cluster after 8 hours so you don't forget it.

## Configure the Backend

```bash
aerolab config backend -t aws -r us-east-1
```

Optional flags:

| Flag | Effect |
|------|--------|
| `-P, --aws.profile` | Use a named AWS CLI profile instead of the default. |
| `--inventory-cache` | Cache resource state locally for faster operations — only if you're the sole user of the account. |
| `--aws.no-public-ip` | See [Gotchas](#gotchas) above. |
| `--skip-pricing` | Skip cost/pricing lookups (still returns instance-type/volume catalogs). |
| `-r` (comma-separated) | Enable multiple regions, e.g. `us-east-1,us-west-2`. |

Verify and check access:

```bash
aerolab config backend
aerolab config backend -t aws --check-access
```

## AWS-Specific Configuration

Instances you create are attached to a security group of your own which allows
every port from the address you are connecting from, and lets those instances
talk to each other, so several people can share one account safely. AeroLab
keeps that group in step with your address as you move between networks. See
[per-user security groups](../commands/config.md#per-user-security-groups) for
how to override the discovered address behind a NAT gateway or VPN, and how to
turn the automatic handling off.

Additional security groups are managed with `config aws`:

```bash
aerolab config aws list-security-groups
aerolab config aws create-security-groups -n aerolab-sg -p 3000-3005
aerolab config aws lock-security-groups -n aerolab-sg -i 203.0.113.7/32
aerolab config aws delete-security-groups -n aerolab-sg
```

## Creating Clusters

Beyond the Quick Start example, common `cluster create` additions:

| Need | Flag(s) |
|------|---------|
| Multiple disks | repeat `--aws.disk type=gp3,size=100,count=3` |
| Specific subnet | `-U subnet-12345678` |
| Custom security group | `--aws.firewall aerolab-sg` |
| Public IP (overrides backend default) | `-L` |
| Spot instances (cheaper) | `--aws.spot` |
| Tags | repeat `--tags Key=Value` |
| EFS volume mount | `--aws.efs-create --aws.efs-mount myefs:/mnt/efs` |

## Resource Expiry Automation

Installs a Lambda that auto-destroys expired resources:

```bash
aerolab config aws expiry-install
aerolab config aws expiry-list
aerolab config aws expiry-run-frequency -f 20   # minutes
aerolab config aws expiry-remove
```

## Next: Lifecycle, Attach, Files, Cleanup

Starting/stopping, controlling the Aerospike service, `attach` (shell/aql/asinfo/asadm),
file upload/download, and cleanup are identical across every backend — see
[Common Operations](common-operations.md). One AWS-specific note: stopping instances
doesn't delete their EBS volumes, so you're still billed until you `destroy` or let expiry
run.

Cluster-level extras: `aerolab cluster add public-ip -n asd` and
`aerolab cluster add firewall -n asd -f aerolab-sg` add these after the fact.

## Troubleshooting

### Credential Issues

```bash
aws sts get-caller-identity
cat ~/.aws/credentials
```

### Region Issues

```bash
aerolab config backend
```

### Permission Issues

```bash
aws iam get-user
```

### Instance Type Availability

```bash
aerolab inventory instance-types
```

### Network Issues

Check security groups, VPC configuration, and subnet settings.

## Next Steps

- [Common Operations](common-operations.md) - lifecycle, attach, files, cleanup
- [Cluster management commands](../commands/cluster.md)
- [AWS-specific volume management](../commands/volumes.md)
- [Aerospike Cloud integration](../cloud/README.md)
