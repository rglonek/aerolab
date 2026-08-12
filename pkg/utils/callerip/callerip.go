// Package callerip resolves the source CIDRs which AeroLab-managed firewalls
// should allow inbound. By default this is the public IP of the machine
// running AeroLab, discovered over HTTP; operators behind a NAT gateway or a
// corporate VPN can override it with an explicit list.
package callerip

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// EnvOverride names the environment variable which replaces public IP
	// discovery with an explicit, comma-separated list of CIDRs.
	EnvOverride = "AEROLAB_FIREWALL_CIDR"
	// DiscoverKeyword is the flag value which asks for the caller IP to be
	// discovered rather than taken literally.
	DiscoverKeyword = "discover-caller-ip"
	// AnyIPv4 opens access to the whole internet. AeroLab never selects this
	// on its own; it is only honoured when set through the override.
	AnyIPv4 = "0.0.0.0/0"

	discoveryURL     = "https://api.ipify.org?format=json"
	discoveryTimeout = 5 * time.Second
)

var (
	lock     sync.Mutex
	override []string
	cached   []string
)

// SetOverride installs an explicit CIDR list, taking precedence over
// discovery. An empty string clears any previously set override. The value is
// a comma-separated list of CIDRs or bare IPv4 addresses.
func SetOverride(cidrs string) error {
	parsed, err := ParseList(cidrs)
	if err != nil {
		return err
	}
	lock.Lock()
	defer lock.Unlock()
	override = parsed
	return nil
}

// Resolve returns the CIDRs which should be allowed inbound access. The
// override is consulted first, then the AEROLAB_FIREWALL_CIDR environment
// variable, and finally the caller's public IP. Successful discovery is
// remembered for the lifetime of the process.
func Resolve() ([]string, error) {
	lock.Lock()
	defer lock.Unlock()
	if len(override) > 0 {
		return append([]string{}, override...), nil
	}
	if env := os.Getenv(EnvOverride); strings.TrimSpace(env) != "" {
		parsed, err := ParseList(env)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", EnvOverride, err)
		}
		if len(parsed) > 0 {
			return parsed, nil
		}
	}
	if len(cached) > 0 {
		return append([]string{}, cached...), nil
	}
	ip, err := discover()
	if err != nil {
		return nil, err
	}
	cidr, err := Normalize(ip)
	if err != nil {
		return nil, fmt.Errorf("discovered caller IP %q is unusable: %w", ip, err)
	}
	cached = []string{cidr}
	return append([]string{}, cached...), nil
}

// Reset discards the memoised discovery result and any installed override.
func Reset() {
	lock.Lock()
	defer lock.Unlock()
	override = nil
	cached = nil
}

// ParseList normalizes a comma-separated list of CIDRs or bare IPv4
// addresses. Entries equal to DiscoverKeyword are dropped, so that a flag
// left at its default does not defeat discovery.
func ParseList(cidrs string) ([]string, error) {
	ret := []string{}
	for _, item := range strings.Split(cidrs, ",") {
		item = strings.TrimSpace(item)
		if item == "" || item == DiscoverKeyword {
			continue
		}
		cidr, err := Normalize(item)
		if err != nil {
			return nil, err
		}
		if !contains(ret, cidr) {
			ret = append(ret, cidr)
		}
	}
	return ret, nil
}

// Normalize converts a bare IPv4 address into a /32 and canonicalizes any
// CIDR by masking off its host bits, which is the form the cloud APIs store
// and therefore the form comparisons must be made in.
func Normalize(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("empty address")
	}
	if !strings.Contains(value, "/") {
		ip := net.ParseIP(value)
		if ip == nil {
			return "", fmt.Errorf("%q is not a valid IP address", value)
		}
		if ip.To4() == nil {
			return "", fmt.Errorf("%q is an IPv6 address; AeroLab firewalls only support IPv4", value)
		}
		return ip.String() + "/32", nil
	}
	ip, network, err := net.ParseCIDR(value)
	if err != nil {
		return "", fmt.Errorf("%q is not a valid CIDR: %w", value, err)
	}
	if ip.To4() == nil {
		return "", fmt.Errorf("%q is an IPv6 CIDR; AeroLab firewalls only support IPv4", value)
	}
	return network.String(), nil
}

// discover asks a public endpoint what our egress IP address is.
func discover() (string, error) {
	client := &http.Client{Timeout: discoveryTimeout}
	resp, err := client.Get(discoveryURL)
	if err != nil {
		return "", fmt.Errorf("could not discover caller public IP: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not discover caller public IP: %s returned %s", discoveryURL, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", fmt.Errorf("could not discover caller public IP: %w", err)
	}
	var payload struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("could not discover caller public IP: %w", err)
	}
	if payload.IP == "" {
		return "", errors.New("could not discover caller public IP: empty response")
	}
	return payload.IP, nil
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
