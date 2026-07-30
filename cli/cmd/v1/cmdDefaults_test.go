package cmd

import (
	"strings"
	"testing"

	flags "github.com/rglonek/go-flags"
)

func newTestParser(t *testing.T) *flags.Parser {
	t.Helper()
	p := flags.NewParser(&Commands{}, flags.HelpFlag|flags.PassDoubleDash)
	ApplyDefaultOverridesToParser(p)
	return p
}

func findCommand(t *testing.T, parser *flags.Parser, path string) *flags.Command {
	t.Helper()
	cmd := parser.Command
	for _, name := range strings.Split(path, "/") {
		cmd = cmd.Find(name)
		if cmd == nil {
			t.Fatalf("command %q not found (stopped at %q)", path, name)
		}
	}
	return cmd
}

func parserDefault(t *testing.T, parser *flags.Parser, path, long string) string {
	t.Helper()
	opt := findCommand(t, parser, path).FindOptionByLongName(long)
	if opt == nil {
		t.Fatalf("command %q has no --%s", path, long)
	}
	if len(opt.Default) != 1 {
		t.Fatalf("command %q --%s has defaults %v, want exactly one", path, long, opt.Default)
	}
	return opt.Default[0]
}

// The default names a user sees are what the flag parser resolves, so assert
// against the parser rather than against the struct tags the parser was built
// from: several of these come from commandDefaultOverrides instead.
func TestDefaultNames(t *testing.T) {
	parser := newTestParser(t)
	tests := []struct {
		path string
		long string
		want string
	}{
		{"cluster/create", "name", DefaultClusterName},
		{"cluster/grow", "name", DefaultClusterName},
		{"cluster/destroy", "name", DefaultClusterName},
		{"aerospike/start", "name", DefaultClusterName},
		{"attach/shell", "name", DefaultClusterName},

		// Pairs of clusters.
		{"xdr/create-clusters", "name", DefaultSourceClusterName},
		{"xdr/create-clusters", "destinations", DefaultDestClusterName},
		{"xdr/connect", "source", DefaultSourceClusterName},
		{"xdr/connect", "destinations", DefaultDestClusterName},
		{"net/block", "source", DefaultSourceClusterName},
		{"net/block", "destination", DefaultDestClusterName},
		{"net/loss-delay", "source", DefaultSourceClusterName},
		{"net/loss-delay", "destination", DefaultDestClusterName},

		// Client groups are named after the client type.
		{"client/create/none", "name", "none-client"},
		{"client/create/base", "name", "base-client"},
		{"client/create/tools", "name", "tools"},
		{"client/create/ams", "name", "ams"},
		{"client/create/vscode", "name", "vscode"},
		{"client/create/graph", "name", "graph"},
		{"client/create/eksctl", "name", "eksctl"},
		{"client/grow/vscode", "name", "vscode"},
		{"client/grow/ams", "name", "ams"},
		{"client/configure/ams", "name", "ams"},
		{"client/configure/tools", "name", "tools"},

		// Commands that are not tied to one client type keep the generic name.
		{"client/destroy", "name", DefaultClientName},
		{"client/configure/firewall", "name", DefaultClientName},
	}
	for _, tc := range tests {
		t.Run(tc.path+"/--"+tc.long, func(t *testing.T) {
			if got := parserDefault(t, parser, tc.path, tc.long); got != tc.want {
				t.Errorf("default = %q, want %q", got, tc.want)
			}
		})
	}
}

// Guards against a command being added (or reverted) with one of the old
// names, which would no longer match anything the other commands create.
func TestNoRetiredDefaultNames(t *testing.T) {
	retired := map[string]bool{"mydc": true, "mydc-xdr": true, "destdc": true}
	var walk func(cmd *flags.Command, path string)
	walk = func(cmd *flags.Command, path string) {
		for _, sub := range cmd.Commands() {
			subPath := sub.Name
			if path != "" {
				subPath = path + "/" + sub.Name
			}
			for _, opt := range allOptions(sub.Group) {
				for _, def := range opt.Default {
					if retired[def] {
						t.Errorf("%s --%s still defaults to the retired name %q", subPath, opt.LongName, def)
					}
				}
			}
			walk(sub, subPath)
		}
	}
	walk(newTestParser(t).Command, "")
}

func allOptions(g *flags.Group) []*flags.Option {
	opts := g.Options()
	for _, sub := range g.Groups() {
		opts = append(opts, allOptions(sub)...)
	}
	return opts
}

// Two client types defaulting to the same group name would make `client create
// a` and `client create b` fight over one group.
func TestClientTypeDefaultNamesAreDistinct(t *testing.T) {
	seen := make(map[string]string, len(clientTypeDefaultNames))
	for clientType, name := range clientTypeDefaultNames {
		if other, dup := seen[name]; dup {
			t.Errorf("client types %q and %q both default to group name %q", other, clientType, name)
		}
		seen[name] = clientType
	}
}

// Every override must land on a real command and a real flag, otherwise it is
// silently doing nothing.
func TestDefaultOverridesApply(t *testing.T) {
	parser := newTestParser(t)
	for path, overrides := range commandDefaultOverrides {
		for long, want := range overrides {
			if got := parserDefault(t, parser, path, long); got != want {
				t.Errorf("%s --%s: default = %q, want %q", path, long, got, want)
			}
		}
	}
}

// The web UI and the MCP schema read defaults off the reflected command tree
// rather than the parser, so the two have to agree.
func TestCommandTreeDefaultsMatchParser(t *testing.T) {
	tree := BuildCommandTree(&Commands{})
	for path, overrides := range commandDefaultOverrides {
		info := findCommandInfo(tree, path)
		if info == nil {
			t.Errorf("command %q missing from the reflected tree", path)
			continue
		}
		for long, want := range overrides {
			var found bool
			for _, p := range info.Parameters {
				if p.Long != long {
					continue
				}
				found = true
				if p.Default != want {
					t.Errorf("%s --%s: tree default = %q, want %q", path, long, p.Default, want)
				}
			}
			if !found {
				t.Errorf("%s: --%s missing from the reflected tree", path, long)
			}
		}
	}
}

func findCommandInfo(root *CommandInfo, path string) *CommandInfo {
	for _, child := range root.Children {
		if child.Path == path {
			return child
		}
		if strings.HasPrefix(path, child.Path+"/") {
			return findCommandInfo(child, path)
		}
	}
	return nil
}
