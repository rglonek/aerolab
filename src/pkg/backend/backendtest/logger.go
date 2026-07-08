// Package backendtest provides reusable test doubles for the backends package:
// a FakeBackend (implementing backends.Backend), a FakeCloud (implementing
// backends.Cloud), inventory fixture builders, and a helper for registering a
// fake Cloud in the global registry with automatic cleanup.
//
// These doubles let unit tests exercise CLI command logic and inventory
// selectors without touching real cloud SDKs, SSH, or Docker.
package backendtest

import "github.com/rglonek/logger"

// QuietLogger returns a logger configured to emit almost nothing (CRITICAL is
// the least-verbose level), keeping test output clean. Tests that want to see
// backend log output can construct their own logger instead.
func QuietLogger() *logger.Logger {
	l := logger.NewLogger()
	l.SetLogLevel(logger.CRITICAL)
	return l
}
