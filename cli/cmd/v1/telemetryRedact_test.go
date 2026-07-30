package cmd

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

type redactNested struct {
	Inner string `long:"inner" telemetry:"redact"`
	Plain string `long:"plain"`
}

type redactGroup struct {
	GroupSecret string `long:"secret" telemetry:"redact"`
}

type redactSample struct {
	Password  string        `short:"p" long:"password" telemetry:"redact"`
	Token     string        `long:"auth-token" telemetry:"redact"`
	Username  string        `long:"user"`
	Count     int           `long:"count"`
	Nested    redactNested  `long:"-"`
	Group     redactGroup   `group:"G" namespace:"gcp"`
	PtrSecret *string       `long:"ptr-secret" telemetry:"redact"`
	List      []string      `long:"list"`
	Secrets   []redactGroup `long:"secrets"`
	hidden    string
	Skipped   string `json:"-"`
}

func TestRedactParamsReplacesTaggedFields(t *testing.T) {
	ptr := "ptr-value"
	in := &redactSample{
		Password:  "hunter2",
		Token:     "tok-abc",
		Username:  "admin",
		Count:     7,
		Nested:    redactNested{Inner: "nested-secret", Plain: "visible"},
		Group:     redactGroup{GroupSecret: "gcp-secret"},
		PtrSecret: &ptr,
		List:      []string{"a", "b"},
		Secrets:   []redactGroup{{GroupSecret: "list-secret"}},
		hidden:    "unexported",
		Skipped:   "json-skipped",
	}

	out, flags := redactParams(in)

	encoded, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("could not marshal the redacted params: %s", err)
	}
	rendered := string(encoded)

	for _, secret := range []string{"hunter2", "tok-abc", "nested-secret", "gcp-secret", "ptr-value", "list-secret", "unexported", "json-skipped"} {
		if strings.Contains(rendered, secret) {
			t.Errorf("redacted output still contains %q: %s", secret, rendered)
		}
	}
	for _, keep := range []string{"admin", "visible", "\"Count\":7"} {
		if !strings.Contains(rendered, keep) {
			t.Errorf("redacted output dropped non-secret value %q: %s", keep, rendered)
		}
	}

	// The original struct must be untouched: the live command still uses it.
	if in.Password != "hunter2" || in.Group.GroupSecret != "gcp-secret" {
		t.Error("redactParams mutated the input struct instead of copying it")
	}

	wantLong := []string{"password", "auth-token", "gcp.secret", "inner", "ptr-secret"}
	for _, name := range wantLong {
		if !flags.long[name] {
			t.Errorf("expected long flag %q to be recorded as redacted, got %v", name, flags.long)
		}
	}
	if !flags.short["p"] {
		t.Errorf("expected short flag \"p\" to be recorded as redacted, got %v", flags.short)
	}
	if flags.long["user"] || flags.long["count"] {
		t.Errorf("non-secret flags were recorded as redacted: %v", flags.long)
	}
}

func TestRedactParamsHandlesNil(t *testing.T) {
	out, flags := redactParams(nil)
	if out != nil {
		t.Errorf("expected nil output for nil params, got %#v", out)
	}
	if !flags.empty() {
		t.Error("expected no redacted flags for nil params")
	}
}

func TestRedactCmdLine(t *testing.T) {
	flags := newRedactedFlags()
	flags.long["password"] = true
	flags.long["gcp.client-secret"] = true
	flags.short["p"] = true

	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "long flag with separate value",
			in:   []string{"data", "insert", "--password", "hunter2", "--ns", "test"},
			want: []string{"data", "insert", "--password", redactedPlaceholder, "--ns", "test"},
		},
		{
			name: "long flag with equals",
			in:   []string{"--password=hunter2"},
			want: []string{"--password=" + redactedPlaceholder},
		},
		{
			name: "namespaced long flag",
			in:   []string{"config", "backend", "--gcp.client-secret=abc123"},
			want: []string{"config", "backend", "--gcp.client-secret=" + redactedPlaceholder},
		},
		{
			name: "short flag with separate value",
			in:   []string{"-p", "hunter2"},
			want: []string{"-p", redactedPlaceholder},
		},
		{
			name: "short flag with attached value",
			in:   []string{"-phunter2"},
			want: []string{"-p" + redactedPlaceholder},
		},
		{
			name: "untouched flags",
			in:   []string{"--user", "admin", "-n", "mydc"},
			want: []string{"--user", "admin", "-n", "mydc"},
		},
		{
			name: "trailing flag with no value",
			in:   []string{"--password"},
			want: []string{"--password"},
		},
		{
			name: "positional args after double dash",
			in:   []string{"attach", "shell", "--", "--password", "hunter2"},
			want: []string{"attach", "shell", "--", "--password", "hunter2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactCmdLine(tt.in, flags)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("redactCmdLine(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsSensitiveDefaultKey(t *testing.T) {
	sensitive := []string{
		"Config.Backend.GCPClientSecret",
		"Cluster.Create.Password",
		"Config.Backend.AWSSecretKey",
		"Agi.Create.SlackToken",
		"Client.Create.EksAwsKeyId",
		"Cloud.Clusters.Create.Credentials",
		"Config.Backend.Username",
	}
	for _, key := range sensitive {
		if !isSensitiveDefaultKey(key) {
			t.Errorf("expected %q to be treated as sensitive", key)
		}
	}

	safe := []string{
		"Config.Backend.Region",
		"Cluster.Create.NodeCount",
		"Config.Backend.Type",
		// A sensitive-looking section name must not drag in its plain fields.
		"Config.Secrets.Region",
	}
	for _, key := range safe {
		if isSensitiveDefaultKey(key) {
			t.Errorf("expected %q to be treated as safe", key)
		}
	}
}

func TestRedactEnvVars(t *testing.T) {
	in := map[string]string{
		"AEROLAB_HOME":           "/home/user/.config/aerolab",
		"AEROLAB_MCP_AUTH_TOKEN": "bearer-secret",
		"AEROLAB_GCP_PASSWORD":   "hunter2",
		"AEROLAB_S3_SECRET_KEY":  "aws-secret",
		"AEROLAB_PROJECT":        "default",
	}
	out := redactEnvVars(in)

	if out["AEROLAB_HOME"] != "/home/user/.config/aerolab" {
		t.Errorf("AEROLAB_HOME was redacted but should not have been: %q", out["AEROLAB_HOME"])
	}
	if out["AEROLAB_PROJECT"] != "default" {
		t.Errorf("AEROLAB_PROJECT was redacted but should not have been: %q", out["AEROLAB_PROJECT"])
	}
	for _, key := range []string{"AEROLAB_MCP_AUTH_TOKEN", "AEROLAB_GCP_PASSWORD", "AEROLAB_S3_SECRET_KEY"} {
		if out[key] != redactedPlaceholder {
			t.Errorf("%s = %q, expected it to be redacted", key, out[key])
		}
	}
}
