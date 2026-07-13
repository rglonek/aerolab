package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	aeroconf "github.com/rglonek/aerospike-config-file-parser"
	flags "github.com/rglonek/go-flags"
	"github.com/rglonek/logger"
)

// writeTempConf writes conf to a temp file and returns its path.
func writeTempConf(t *testing.T, conf string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "aerospike.conf")
	if err := os.WriteFile(p, []byte(conf), 0644); err != nil {
		t.Fatalf("write temp conf: %v", err)
	}
	return p
}

func confHasVal(t *testing.T, cfg aeroconf.Stanza, parts []string, key string) bool {
	t.Helper()
	s := cfg
	for _, p := range parts {
		if s.Type(p) != aeroconf.ValueStanza {
			return false
		}
		s = s.Stanza(p)
	}
	vals, err := s.GetValues(key)
	return err == nil && len(vals) > 0 && vals[0] != nil
}

// TestInspectCustomConfig_AugmentsMissingBasics verifies that a namespace-only
// config gets every missing basic item injected and that the result parses back
// into a complete, self-consistent config.
func TestInspectCustomConfig_AugmentsMissingBasics(t *testing.T) {
	const conf = `namespace test {
    replication-factor 1
    strong-consistency true
    storage-engine device {
        file /opt/aerospike/data/test.dat
        filesize 4G
    }
}
`
	report, err := inspectCustomConfig([]byte(conf))
	if err != nil {
		t.Fatalf("inspectCustomConfig: %v", err)
	}
	if !report.hasNamespace {
		t.Errorf("expected hasNamespace=true")
	}
	if report.nonDefaultPort != "" {
		t.Errorf("expected no nonDefaultPort, got %q", report.nonDefaultPort)
	}
	wantMissing := []string{
		"service (proto-fd-max 15000)",
		"logging (file /var/log/aerospike.log)",
		"network.service.port (3000)",
		"network.heartbeat (mode mesh, port 3002)",
		"network.fabric.port (3001)",
	}
	if len(report.missing) != len(wantMissing) {
		t.Fatalf("missing mismatch:\n got %v\nwant %v", report.missing, wantMissing)
	}
	for i := range wantMissing {
		if report.missing[i] != wantMissing[i] {
			t.Fatalf("missing[%d] = %q, want %q", i, report.missing[i], wantMissing[i])
		}
	}

	// the augmented config must parse and contain all the injected basics
	cfg, err := aeroconf.Parse(bytes.NewReader(report.augmented))
	if err != nil {
		t.Fatalf("augmented config does not parse: %v\n%s", err, report.augmented)
	}
	if !confHasVal(t, cfg, []string{"service"}, "proto-fd-max") {
		t.Error("augmented config missing service.proto-fd-max")
	}
	if cfg.Type("logging") != aeroconf.ValueStanza {
		t.Error("augmented config missing logging stanza")
	}
	if !confHasVal(t, cfg, []string{"network", "service"}, "port") {
		t.Error("augmented config missing network.service.port")
	}
	if !confHasVal(t, cfg, []string{"network", "heartbeat"}, "mode") || !confHasVal(t, cfg, []string{"network", "heartbeat"}, "port") {
		t.Error("augmented config missing network.heartbeat mode/port")
	}
	if !confHasVal(t, cfg, []string{"network", "fabric"}, "port") {
		t.Error("augmented config missing network.fabric.port")
	}
	// the original namespace must survive augmentation
	if cfg.Type("namespace test") != aeroconf.ValueStanza {
		t.Error("augmented config dropped the original namespace")
	}
	// the injected service port must be the default 3000
	if vals, _ := cfg.Stanza("network").Stanza("service").GetValues("port"); len(vals) == 0 || vals[0] == nil || *vals[0] != "3000" {
		t.Errorf("expected injected service port 3000, got %v", vals)
	}
}

// TestInspectCustomConfig_CompleteConfig verifies a config with all basics
// yields no missing items and no augmentation.
func TestInspectCustomConfig_CompleteConfig(t *testing.T) {
	const conf = `service {
    proto-fd-max 15000
}
logging {
    file /var/log/aerospike.log {
        context any info
    }
}
network {
    service {
        port 3000
    }
    heartbeat {
        mode mesh
        port 3002
    }
    fabric {
        port 3001
    }
}
namespace test {
    replication-factor 1
    storage-engine memory {
        data-size 1G
    }
}
`
	report, err := inspectCustomConfig([]byte(conf))
	if err != nil {
		t.Fatalf("inspectCustomConfig: %v", err)
	}
	if len(report.missing) != 0 {
		t.Errorf("expected no missing items, got %v", report.missing)
	}
	if report.augmented != nil {
		t.Errorf("expected no augmentation for a complete config")
	}
	if !report.hasNamespace {
		t.Errorf("expected hasNamespace=true")
	}
}

// TestInspectCustomConfig_NonDefaultPort verifies the non-3000 port is surfaced
// and not mistaken for a missing port.
func TestInspectCustomConfig_NonDefaultPort(t *testing.T) {
	const conf = `network {
    service {
        port 3100
    }
}
namespace test {
    replication-factor 1
    storage-engine memory {
        data-size 1G
    }
}
`
	report, err := inspectCustomConfig([]byte(conf))
	if err != nil {
		t.Fatalf("inspectCustomConfig: %v", err)
	}
	if report.nonDefaultPort != "3100" {
		t.Errorf("expected nonDefaultPort=3100, got %q", report.nonDefaultPort)
	}
	for _, m := range report.missing {
		if m == "network.service.port (3000)" {
			t.Error("service.port present but reported missing")
		}
	}
}

// TestCheckCustomConfigBasics_NonInteractive verifies that, when not running
// interactively, checkCustomConfigBasics only warns (never errors) and never
// mutates the config in-memory.
func TestCheckCustomConfigBasics_NonInteractive(t *testing.T) {
	// IsInteractive() is false under `go test`, so this exercises the
	// warn-only path.
	c := &ClusterCreateCmd{}
	c.CustomConfigFilePath = flags.Filename(writeTempConf(t, "namespace test {\n    replication-factor 1\n    storage-engine memory {\n        data-size 1G\n    }\n}\n"))
	if err := c.checkCustomConfigBasics(logger.NewLogger()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.customConfig) != 0 {
		t.Fatalf("customConfig should stay empty in non-interactive mode, got %d bytes", len(c.customConfig))
	}
}

// TestCheckCustomConfigBasics_NoCustomConfig verifies the no-op path.
func TestCheckCustomConfigBasics_NoCustomConfig(t *testing.T) {
	c := &ClusterCreateCmd{}
	if err := c.checkCustomConfigBasics(logger.NewLogger()); err != nil {
		t.Fatalf("unexpected error with no custom config: %v", err)
	}
}
