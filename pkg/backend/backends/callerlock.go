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

// SSHPort is the well-known SSH port. It is still used by tools that talk
// about SSH specifically (legacy warnings, integration checks), but the
// per-user caller firewall now opens every port, not only this one.
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
	r = CanonicalCallerRule(r)
	if r.Protocol == ProtocolAll && r.FromPort == -1 {
		return fmt.Sprintf("all from %s", r.Cidr)
	}
	return fmt.Sprintf("%s:%d-%d from %s", r.Protocol, r.FromPort, r.ToPort, r.Cidr)
}

// CallerSSHPorts is the historical SSH-only set. Prefer CallerAllPorts for
// the per-user default firewall.
func CallerSSHPorts() []CallerPort {
	return []CallerPort{{Protocol: ProtocolTCP, FromPort: SSHPort, ToPort: SSHPort}}
}

// CallerAllPorts opens every protocol and port from the caller. The per-user
// firewall is already locked to the caller's address, so listing individual
// service ports (AMS 3000, Grafana 8080, and the rest) is unnecessary.
func CallerAllPorts() []CallerPort {
	return []CallerPort{{Protocol: ProtocolAll, FromPort: -1, ToPort: -1}}
}

// CanonicalCallerRule folds equivalent "all protocols" encodings onto one
// shape. AWS describes protocol -1 with omitted (zero) ports; GCP and our
// create path use FromPort/ToPort -1.
func CanonicalCallerRule(r CallerRule) CallerRule {
	if r.Protocol == ProtocolAll || r.Protocol == "all" {
		r.Protocol = ProtocolAll
		r.FromPort = -1
		r.ToPort = -1
	}
	return r
}

// BuildCallerRules expands port ranges across source CIDRs, producing the full
// set of rules the caller's firewall should carry.
func BuildCallerRules(ports []CallerPort, cidrs []string) []CallerRule {
	rules := []CallerRule{}
	for _, port := range ports {
		for _, cidr := range cidrs {
			rule := CanonicalCallerRule(CallerRule{
				Protocol: port.Protocol,
				FromPort: port.FromPort,
				ToPort:   port.ToPort,
				Cidr:     cidr,
			})
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
	have := make([]CallerRule, 0, len(existing))
	for _, r := range existing {
		have = append(have, CanonicalCallerRule(r))
	}
	want := make([]CallerRule, 0, len(desired))
	for _, r := range desired {
		want = append(want, CanonicalCallerRule(r))
	}
	for _, w := range want {
		if !slices.Contains(have, w) && !slices.Contains(add, w) {
			add = append(add, w)
		}
	}
	for _, h := range have {
		if !slices.Contains(want, h) && !slices.Contains(remove, h) {
			remove = append(remove, h)
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
