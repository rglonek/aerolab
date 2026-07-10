//go:build !embedexpiry

package backends

// ExpiryBinary is empty in builds without -tags=embedexpiry (e.g. when
// aerolab is consumed as a library via `go get`). Expiry deployment paths
// check for this and return ErrNoExpiryBinary. Official builds use the
// Makefile, which always sets the tag.
var ExpiryBinary []byte
