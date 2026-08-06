//go:build integration_docker

package goproxy_test

import (
	"testing"

	"github.com/aerospike/aerolab/pkg/utils/installers/goproxy"
	"github.com/aerospike/aerolab/tests/installers/installertest"
	"github.com/stretchr/testify/require"
)

func TestGoproxyLatestUbuntu24(t *testing.T) {
	installertest.RequireDocker(t)

	script, err := goproxy.GetLinuxInstallScript(nil, nil, false)
	require.NoError(t, err)
	require.NotNil(t, script)
	require.NotEmpty(t, script)

	installertest.RunScriptInImage(t, "{arch}/ubuntu:24.04", script)
}

func TestGoproxyLatestCentos8(t *testing.T) {
	installertest.RequireDocker(t)

	script, err := goproxy.GetLinuxInstallScript(nil, nil, false)
	require.NoError(t, err)
	require.NotNil(t, script)
	require.NotEmpty(t, script)

	installertest.RunScriptInImage(t, "quay.io/centos/{arch}:stream8", script)
}

func TestGoproxyLatestStable(t *testing.T) {
	installertest.RequireDocker(t)

	script, err := goproxy.GetLinuxInstallScript(nil, new(false), false)
	require.NoError(t, err)
	require.NotNil(t, script)
	require.NotEmpty(t, script)

	installertest.RunScriptInImage(t, "{arch}/ubuntu:24.04", script)
}
