package backends

// Identity describes who is running AeroLab and which source addresses their
// managed firewalls should allow inbound.
//
// It is populated by the CLI. In-cloud consumers of this package - the expiry
// lambda and the AGI processes - leave it unset, which disables all caller-IP
// firewall handling: their "caller IP" would be a machine inside the cloud,
// not the person operating AeroLab.
type Identity struct {
	// Owner is the sanitized username the current caller's resources are
	// tagged with. Per-call owners, where available, take precedence.
	Owner string
	// CallerCidrs resolves the source CIDRs which the caller's firewalls
	// should allow. A nil function disables caller-IP firewall handling
	// entirely.
	CallerCidrs func() ([]string, error)
	// Autolock enables creating, re-locking and attaching the caller's
	// firewall as a side effect of commands which touch instances.
	Autolock bool
}

// CanLock reports whether the identity carries enough information to manage
// caller-locked firewalls.
func (i *Identity) CanLock() bool {
	return i != nil && i.Owner != "" && i.CallerCidrs != nil
}

// AutolockEnabled reports whether firewalls may be reconciled and attached as
// a side effect of other commands.
func (i *Identity) AutolockEnabled() bool {
	return i.CanLock() && i.Autolock
}

// Cidrs resolves the caller's source CIDRs, or returns nil when no identity is
// configured.
func (i *Identity) Cidrs() ([]string, error) {
	if !i.CanLock() {
		return nil, nil
	}
	return i.CallerCidrs()
}

// OwnerOr returns the given per-call owner when set, falling back to the
// identity's owner. WebUI runs jobs for different users against one backend
// instance, so a per-call owner is always more accurate than the process-wide
// one.
func (i *Identity) OwnerOr(owner string) string {
	if owner != "" {
		return owner
	}
	if i == nil {
		return ""
	}
	return i.Owner
}
