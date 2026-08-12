package backends

import (
	"fmt"
	"slices"
	"strings"
)

// CallerRuleDescription marks the ingress rules AeroLab manages on behalf of
// the caller. Only rules carrying this marker are ever revoked when the
// caller's address changes, so rules added by hand, or by
// 'config aws|gcp create-security-groups', survive untouched.
const CallerRuleDescription = "aerolab-caller-ip"

// FirewallRoleDefault marks a firewall as the per-user group AeroLab attaches
// to every instance that user works with.
const FirewallRoleDefault = "default"

// FirewallRoleInternal marks a firewall which only allows instances of the
// same group to talk to each other.
const FirewallRoleInternal = "internal"

// FirewallRoleAGI marks a firewall serving AGI's web ports.
const FirewallRoleAGI = "agi"

// SSHPort is the port the caller must be able to reach in order to work with
// an instance at all.
const SSHPort = 22

// AnyIPv4Cidr allows the whole internet in. AeroLab never selects it on its
// own; it appears only where an operator asked for it explicitly, or on
// security groups left behind by older versions.
const AnyIPv4Cidr = "0.0.0.0/0"

// CallerPort is a port range which should be reachable from the caller.
type CallerPort struct {
	Protocol string
	FromPort int
	ToPort   int
}

// CallerRule is a single source CIDR allowed into a single port range.
type CallerRule struct {
	Protocol string
	FromPort int
	ToPort   int
	Cidr     string
}

// String renders the rule for logging.
func (r CallerRule) String() string {
	return fmt.Sprintf("%s:%d-%d from %s", r.Protocol, r.FromPort, r.ToPort, r.Cidr)
}

// CallerSSHPorts is the minimum a caller needs to reach an instance.
func CallerSSHPorts() []CallerPort {
	return []CallerPort{{Protocol: ProtocolTCP, FromPort: SSHPort, ToPort: SSHPort}}
}

// BuildCallerRules expands port ranges across source CIDRs, producing the full
// set of rules the caller's firewall should carry.
func BuildCallerRules(ports []CallerPort, cidrs []string) []CallerRule {
	rules := []CallerRule{}
	for _, port := range ports {
		for _, cidr := range cidrs {
			rule := CallerRule{
				Protocol: port.Protocol,
				FromPort: port.FromPort,
				ToPort:   port.ToPort,
				Cidr:     cidr,
			}
			if !slices.Contains(rules, rule) {
				rules = append(rules, rule)
			}
		}
	}
	return rules
}

// DiffCallerRules compares the AeroLab-managed rules a firewall currently
// carries against the ones it should carry, returning the rules to authorise
// and the ones to revoke. Only rules present in existing are ever proposed for
// removal, so callers must pass in only the rules they own.
func DiffCallerRules(existing []CallerRule, desired []CallerRule) (add []CallerRule, remove []CallerRule) {
	add = []CallerRule{}
	remove = []CallerRule{}
	for _, want := range desired {
		if !slices.Contains(existing, want) && !slices.Contains(add, want) {
			add = append(add, want)
		}
	}
	for _, have := range existing {
		if !slices.Contains(desired, have) && !slices.Contains(remove, have) {
			remove = append(remove, have)
		}
	}
	return add, remove
}

// DescribeCallerRules renders a rule set for logging.
func DescribeCallerRules(rules []CallerRule) string {
	out := make([]string, 0, len(rules))
	for _, rule := range rules {
		out = append(out, rule.String())
	}
	return strings.Join(out, ", ")
}
