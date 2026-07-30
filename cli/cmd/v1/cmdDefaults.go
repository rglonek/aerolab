package cmd

import (
	"strings"

	flags "github.com/rglonek/go-flags"
)

// The naming convention behind the `default` struct tags on cluster and client
// name parameters. Struct tags have to spell the value out, so these constants
// are what the tags are checked against (see TestDefaultNames) rather than
// their source.
const (
	// DefaultClusterName is the cluster every single-cluster command works on
	// unless told otherwise.
	DefaultClusterName = "asd"
	// DefaultSourceClusterName and DefaultDestClusterName name the two ends of
	// commands that take a pair of clusters, such as xdr create-clusters,
	// xdr connect and net block.
	DefaultSourceClusterName = DefaultClusterName + "-source"
	DefaultDestClusterName   = DefaultClusterName + "-dest"
	// DefaultClientName is the group name for client commands that are not
	// specific to one client type.
	DefaultClientName = "client"
)

// commandDefaultOverrides holds defaults that cannot be written as a struct
// tag because the parameter is declared on a struct that several commands
// share. `client create vscode` and `client create ams` both take -n from
// ClientCreateNoneCmd, and `xdr create-clusters` takes -n from
// ClusterCreateCmd, yet each wants a different default.
//
// Keyed by command path (slash separated, as the web UI spells it) and then by
// the parameter's long name. Everything that presents a default to a user --
// the flag parser, the web UI, the MCP schema and the "equivalent command"
// lines -- resolves it through here, so a command listed below behaves as if
// the value had been written in its own struct tag.
var commandDefaultOverrides = map[string]map[string]string{
	"xdr/create-clusters": {"name": DefaultSourceClusterName},
}

// clientTypeDefaultNames is the default group name for each client type.
// Naming a client group after what is installed on it means the groups created
// by two different `client create` subcommands do not collide, which they did
// when every type defaulted to "client". The bare type name is used where it
// reads as a name on its own; "none" and "base" do not, so they are suffixed.
var clientTypeDefaultNames = map[string]string{
	"none":   "none-client",
	"base":   "base-client",
	"tools":  "tools",
	"ams":    "ams",
	"vscode": "vscode",
	"graph":  "graph",
	"eksctl": "eksctl",
}

func init() {
	// `client create X` and `client grow X` are the same struct, so both need
	// the type's default name.
	for clientType, name := range clientTypeDefaultNames {
		for _, verb := range []string{"create", "grow"} {
			commandDefaultOverrides["client/"+verb+"/"+clientType] = map[string]string{"name": name}
		}
	}
}

// DefaultOverridesForPath returns the default overrides for a command, or nil
// if it has none. The path may be given slash separated ("client/create/ams")
// or as the components of the command line.
func DefaultOverridesForPath(path ...string) map[string]string {
	return commandDefaultOverrides[strings.Join(path, "/")]
}

// ApplyDefaultOverridesToParser rewrites the defaults of the parser's options
// to match commandDefaultOverrides. It must run before the arguments are
// parsed, since go-flags applies defaults at the end of a parse.
func ApplyDefaultOverridesToParser(parser *flags.Parser) {
	if parser == nil {
		return
	}
	applyDefaultOverridesToCommand(parser.Command, "")
}

func applyDefaultOverridesToCommand(command *flags.Command, parentPath string) {
	if command == nil {
		return
	}
	for _, sub := range command.Commands() {
		path := sub.Name
		if parentPath != "" {
			path = parentPath + "/" + sub.Name
		}
		for long, def := range commandDefaultOverrides[path] {
			if opt := sub.FindOptionByLongName(long); opt != nil {
				opt.Default = []string{def}
			}
		}
		applyDefaultOverridesToCommand(sub, path)
	}
}
