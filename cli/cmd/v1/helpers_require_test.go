package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The Require* helpers only prompt when aerolab is interactive. Everywhere
// else (CI, the Web UI, MCP, piped output) they must keep returning the same
// "<name> is required" error the commands returned before prompting existed.
func TestRequireHelpersErrorWhenNotInteractive(t *testing.T) {
	t.Setenv("AEROLAB_NONINTERACTIVE", "1")
	require.False(t, IsInteractive())

	_, err := RequireString("", "cluster name")
	require.EqualError(t, err, "cluster name is required")

	_, err = RequireSecret("", "password")
	require.EqualError(t, err, "password is required")

	_, err = RequireInt(0, "cluster size")
	require.EqualError(t, err, "cluster size is required")

	_, err = RequireFloat(0, "mounts size limit pct")
	require.EqualError(t, err, "mounts size limit pct is required")

	_, err = RequireChoice("", "data storage", "memory", "local-disk")
	require.EqualError(t, err, "data storage is required")
}

func TestRequireHelpersPassThroughProvidedValues(t *testing.T) {
	t.Setenv("AEROLAB_NONINTERACTIVE", "1")

	s, err := RequireString("mydc", "cluster name")
	require.NoError(t, err)
	require.Equal(t, "mydc", s)

	s, err = RequireSecret("hunter2", "password")
	require.NoError(t, err)
	require.Equal(t, "hunter2", s)

	i, err := RequireInt(3, "cluster size")
	require.NoError(t, err)
	require.Equal(t, 3, i)

	f, err := RequireFloat(90, "mounts size limit pct")
	require.NoError(t, err)
	require.Equal(t, float64(90), f)

	s, err = RequireChoice("memory", "data storage", "memory", "local-disk")
	require.NoError(t, err)
	require.Equal(t, "memory", s)
}

// A choice with nothing to choose from cannot be resolved even interactively,
// so it must fall back to the required-option error rather than showing an
// empty picker.
func TestRequireChoiceWithNoOptions(t *testing.T) {
	_, err := RequireChoice("", "firewall name")
	require.EqualError(t, err, "firewall name is required")
}
