// Package agi provides shared constants and utilities for the Aerospike Grafana Integration.
package agi

import (
	_ "embed"
)

// agiproxy.tgz is committed so this package is importable without a generate
// step; gzip -n keeps the archive byte-stable across regenerations.
//go:generate sh -c "cd ../../web/agiproxy && tar -cf - * | gzip -n > ../../pkg/agi/agiproxy.tgz"

// AgiProxyWeb contains the embedded web UI assets for the AGI proxy.
// This tarball is generated from web/agiproxy/ and contains:
//   - index.html - Template with {{.HTTPTitle}}, {{.Title}}, {{.Description}} variables
//   - dist/ - Static CSS and JavaScript assets (bootstrap, datatables, fontawesome, etc.)
//
//go:embed agiproxy.tgz
var AgiProxyWeb []byte

