//go:build embedexpiry

package backends

import (
	_ "embed"
)

// ExpiryBinary is the compiled expiry function deployment package
// (expiry.linux.amd64.zip, produced by pkg/expiry/compile.sh via go generate).
// It is only embedded when building with -tags=embedexpiry so that the module
// remains importable from a fresh checkout / module proxy download, where the
// generated zip does not exist.
//
//go:embed expiry.linux.amd64.zip
var ExpiryBinary []byte
