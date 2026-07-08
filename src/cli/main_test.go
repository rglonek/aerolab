package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	os.RemoveAll("./testdata")
	t.Setenv("AEROLAB_HOME", "./testdata")
	// Keep the test hermetic: disable the background upgrade check and telemetry
	// so no lingering goroutine races with the os.Stdout/os.Stderr restore below.
	t.Setenv("AEROLAB_TEST", "1")
	t.Setenv("AEROLAB_TELEMETRY_DISABLE", "1")
	defer os.RemoveAll("./testdata")
	os.Args = []string{"aerolab", "version"}
	var buf bytes.Buffer
	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStdout := os.Stdout
	origStderr := os.Stderr
	os.Stdout = w
	os.Stderr = w
	// Copy the pipe into buf on a goroutine and signal completion, so buf is
	// only read from the main goroutine after the copy has fully finished.
	done := make(chan struct{})
	go func() {
		io.Copy(&buf, r) //nolint:errcheck
		close(done)
	}()
	run([]string{"version"}) //nolint:errcheck
	w.Close()
	<-done
	os.Stdout = origStdout
	os.Stderr = origStderr
	_, err = os.Stat("./testdata")
	require.NoError(t, err)
	vString := buf.String()
	require.Equal(t, true, strings.HasPrefix(vString, "v"))
	require.Equal(t, true, strings.HasSuffix(vString, "-unofficial\n"))
}
