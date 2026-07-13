package cmd

import (
	flags "github.com/rglonek/go-flags"
)

// Client backend flag structs.
//
// Historically (aerolab v7) the `client create ...` commands shared the exact
// same backend flags as `cluster create` (they embedded the clusterCreateCmd*
// structs). aerolab v8 briefly switched the client commands to the newer,
// namespaced `instances create` backend flags (e.g. `--aws.instance`,
// `--gcp.no-public-ip`), which broke every documented v7 `client create`
// invocation.
//
// To keep breakage to an absolute minimum we mirror the `cluster create`
// (v7-compatible) flag naming here: no `aws.`/`gcp.`/`docker.` namespace and
// the original long flag names (`--instance-type`, `--ami`, `--aws-disk`,
// `--aws-spot-instance`, `--public-ip`, ...). The Go field names are kept
// identical to the InstancesCreateCmd* structs so the rest of the client code
// continues to compile unchanged; toInstances() converts to the instances
// representation actually consumed by InstancesCreateCmd.

// ClientCreateCmdAws mirrors the v7/cluster-create AWS backend flags for the
// `client create` commands.
type ClientCreateCmdAws struct {
	ImageID            string          `short:"A" long:"ami" description:"Custom AMI/image ID to use for the instances; ignores OS, Version, Arch"`
	InstanceType       guiInstanceType `short:"I" long:"instance-type" description:"instance type to use" webchoice:"method::List"`
	Disks              []string        `long:"aws-disk" description:"EBS disks, format: type={gp3|gp2|io2|io1},size={GB}[,iops={cnt}][,throughput={mb/s}][,count=5][,encrypted=true|false]; first specified volume is the root volume, all subsequent volumes are additional attached volumes" default:"type=gp3,size=20"`
	NetworkPlacement   string          `short:"U" long:"subnet-id" description:"network placement: region name, VPC-ID or subnet-ID; empty=default at first region"`
	PublicIP           bool            `short:"L" long:"public-ip" description:"if set, force a public IP to be assigned to the instances even if the backend is configured to disable public IPs"`
	Firewalls          []string        `long:"secgroup-name" description:"Extra security group names to assign to the instances"`
	SpotInstance       bool            `long:"aws-spot-instance" description:"set to request a spot instance in place of on-demand"`
	Expire             TypeExpiry      `long:"aws-expire" description:"length of life of nodes prior to expiry; Y/M/W/D/h/m/s, ex 1D12h 2W 1Y6M; 0: no expiry" default:"30h"`
	IAMInstanceProfile string          `long:"aws-instance-profile" description:"IAM instance profile to use for the instances"`
	CustomDNS          InstanceDNS     `group:"Automated Custom Route53 DNS" namespace:"aws" description:"backend-aws"`
}

func (a *ClientCreateCmdAws) toInstances() InstancesCreateCmdAws {
	return InstancesCreateCmdAws{
		ImageID:            a.ImageID,
		Expire:             a.Expire,
		NetworkPlacement:   a.NetworkPlacement,
		InstanceType:       a.InstanceType,
		Disks:              a.Disks,
		Firewalls:          a.Firewalls,
		SpotInstance:       a.SpotInstance,
		IAMInstanceProfile: a.IAMInstanceProfile,
		CustomDNS:          a.CustomDNS,
		// DisablePublicIP is derived from backend config + PublicIP by the caller.
	}
}

// ClientCreateCmdGcp mirrors the v7/cluster-create GCP backend flags for the
// `client create` commands.
type ClientCreateCmdGcp struct {
	ImageName          string          `long:"image" description:"Custom source image to use for the instances; ignores OS, Version, Arch; format: projects/<project>/global/images/<image>"`
	InstanceType       guiInstanceType `long:"instance" description:"instance type to use" webchoice:"method::List"`
	Disks              []string        `long:"gcp-disk" description:"disks, format: type={pd-*,hyperdisk-*,local-ssd}[,size={GB}][,iops={cnt}][,throughput={mb/s}][,count=5]; first specified volume is the root volume, cannot be local-ssd" default:"type=pd-ssd,size=20"`
	PublicIP           bool            `long:"external-ip" description:"if set, force a public IP to be assigned to the instances even if the backend is configured to disable public IPs"`
	Zone               guiZone         `long:"zone" description:"zone name to deploy to; empty=default at first region" webchoice:"method::List"`
	VPC                guiVpc          `long:"vpc" description:"VPC network name to use; empty=default VPC" webchoice:"method::List"`
	Subnet             string          `long:"subnet" description:"GCP subnet name within the selected VPC; empty=auto-select first subnet in the zone's region"`
	Firewalls          []string        `long:"firewall" description:"Extra firewall names to assign to the instances"`
	SpotInstance       bool            `long:"gcp-spot-instance" description:"set to request a spot instance in place of on-demand"`
	Expire             TypeExpiry      `long:"gcp-expire" description:"length of life of nodes prior to expiry; Y/M/W/D/h/m/s, ex 1D12h 2W 1Y6M; 0: no expiry" default:"30h"`
	IAMInstanceProfile string          `long:"gcp-instance-profile" description:"IAM instance profile to use for the instances"`
	MinCPUPlatform     string          `long:"gcp-min-cpu-platform" description:"set the minimum CPU platform; see https://cloud.google.com/compute/docs/instances/specify-min-cpu-platform"`
	GVNIC              bool            `long:"gcp-gvnic" description:"use Google Virtual NIC (gVNIC) instead of the default VirtIO NIC; required for highest network performance and for some newer instance types"`
	OnHostMaintenance  string          `long:"on-host-maintenance-policy" description:"on-host maintenance policy: MIGRATE or TERMINATE; defaults to MIGRATE (or TERMINATE for spot and GPU instance types, e.g. A2/A3/A4/G2)"`
	CustomDNS          InstanceDNS     `group:"Automated Custom GCP DNS" namespace:"gcp" description:"backend-gcp"`
}

func (g *ClientCreateCmdGcp) toInstances() InstancesCreateCmdGcp {
	return InstancesCreateCmdGcp{
		ImageName:          g.ImageName,
		Expire:             g.Expire,
		Zone:               g.Zone,
		VPC:                g.VPC,
		Subnet:             g.Subnet,
		InstanceType:       g.InstanceType,
		Disks:              g.Disks,
		Firewalls:          g.Firewalls,
		SpotInstance:       g.SpotInstance,
		IAMInstanceProfile: g.IAMInstanceProfile,
		MinCPUPlatform:     g.MinCPUPlatform,
		GVNIC:              g.GVNIC,
		OnHostMaintenance:  g.OnHostMaintenance,
		CustomDNS:          g.CustomDNS,
		// DisablePublicIP is derived from backend config + PublicIP by the caller.
	}
}

// ClientCreateCmdDocker mirrors the v7/cluster-create Docker backend flags for
// the `client create` commands. It keeps the extra v8-only knobs (custom image,
// registry auth, advanced config, ...) that the graph/ams/vscode clients rely
// on, under names that do not collide with the AWS/GCP flags once flattened.
type ClientCreateCmdDocker struct {
	ImageName          string         `long:"docker-image" description:"Custom image name to use for the instances; ignores OS, Version, Arch"`
	NetworkName        string         `long:"network" description:"specify a network name to use for non-default docker network"`
	Disks              []string       `long:"docker-disk" description:"Format: {volumeName|/hostPath}:{mountTargetDirectory}[:ro|:rw]; example: volume1:/mnt/data or /host/path:/container/path:ro"`
	ExposePorts        []string       `long:"expose-ports" description:"Format: [+]{hostPort}:{containerPort}; example: 8080:80 or +8080:80; + maps to next available port; can be specified multiple times"`
	StopTimeout        *int           `long:"stop-timeout" description:"Container default stop timeout in seconds before force-stop"`
	Privileged         bool           `long:"privileged" description:"Docker only: run container in privileged mode"`
	RestartPolicy      string         `long:"restart" description:"Container restart policy: Always, None, OnFailure, UnlessStopped"`
	MaxRestartRetries  int            `long:"max-restart-retries" description:"Maximum number of restart attempts"`
	ShmSize            int64          `long:"shm-size" description:"Size of /dev/shm in bytes"`
	CpuLimit           string         `long:"cpu-limit" description:"Impose CPU speed limit. Values acceptable could be '1' or '2' or '0.5' etc."`
	RamLimit           string         `long:"ram-limit" description:"Limit RAM available to each node, e.g. 500m, or 1g."`
	SwapLimit          string         `long:"swap-limit" description:"Limit the amount of total memory (ram+swap) each node can use, e.g. 600m. If ram-limit==swap-limit, no swap is available."`
	AdvancedConfigPath flags.Filename `long:"advanced-config" description:"Path to JSON file containing advanced Docker container configuration"`
	RegistryUser       string         `long:"registry-user" description:"Username for docker registry authentication when pulling custom images"`
	RegistryPass       string         `long:"registry-pass" description:"Password for docker registry authentication when pulling custom images" webtype:"password"`
	RegistryURL        string         `long:"registry-url" description:"Registry URL (e.g., docker.io, ghcr.io); if empty, uses default registry"`
}

func (d *ClientCreateCmdDocker) toInstances() InstancesCreateCmdDocker {
	return InstancesCreateCmdDocker{
		ImageName:          d.ImageName,
		NetworkName:        d.NetworkName,
		Disks:              d.Disks,
		ExposePorts:        d.ExposePorts,
		StopTimeout:        d.StopTimeout,
		Privileged:         d.Privileged,
		RestartPolicy:      d.RestartPolicy,
		MaxRestartRetries:  d.MaxRestartRetries,
		ShmSize:            d.ShmSize,
		CpuLimit:           d.CpuLimit,
		RamLimit:           d.RamLimit,
		SwapLimit:          d.SwapLimit,
		AdvancedConfigPath: d.AdvancedConfigPath,
		RegistryUser:       d.RegistryUser,
		RegistryPass:       d.RegistryPass,
		RegistryURL:        d.RegistryURL,
	}
}
