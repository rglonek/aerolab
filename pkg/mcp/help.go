package mcp

import (
	"bytes"
	"errors"

	flags "github.com/rglonek/go-flags"
)

// HelpRenderer renders the CLI help text for a command path (slash- or
// space-separated path segments). Implementations must not write to the
// real stdout/stderr or call os.Exit.
type HelpRenderer func(path []string) (string, error)

// OptionsFactory returns a freshly allocated root options struct, typically
// &cmd.Commands{}. Using a factory lets pkg/mcp render help without
// importing the cmd package (which would create a circular dependency).
type OptionsFactory func() any

// ParserHook adjusts a freshly built parser before it renders help, so the
// help text matches what the CLI itself would print. The cmd package uses it
// to apply the per-command default overrides it cannot express as struct tags.
type ParserHook func(*flags.Parser)

// RenderHelpFromFactory returns a HelpRenderer that instantiates a fresh
// options struct per call, parses the command path with go-flags, and
// captures the parser's help output to a buffer.
//
// Unlike cmd.PrintHelp, this path never calls os.Exit and never prints to the
// real stdout/stderr. It is safe to call concurrently.
//
// Behavior:
//   - An empty path produces root-level help.
//   - Paths pointing to a subcommand produce that subcommand's help.
//   - Unknown paths fall back to root help with no error (useful for tool
//     descriptions).
func RenderHelpFromFactory(factory OptionsFactory, hooks ...ParserHook) HelpRenderer {
	newParser := func(opts any) *flags.Parser {
		// HelpFlag produces ErrHelp when -h is encountered (we rely on that
		// to set the active subcommand). PassDoubleDash keeps positional
		// args intact. We intentionally omit PrintErrors so go-flags does
		// not touch os.Stderr.
		p := flags.NewParser(opts, flags.HelpFlag|flags.PassDoubleDash)
		for _, hook := range hooks {
			if hook != nil {
				hook(p)
			}
		}
		return p
	}
	return func(path []string) (string, error) {
		if factory == nil {
			return "", errors.New("mcp: help renderer missing options factory")
		}
		opts := factory()
		if opts == nil {
			return "", errors.New("mcp: help renderer factory returned nil")
		}
		parser := newParser(opts)

		args := append(append([]string{}, path...), "-h")
		if _, err := parser.ParseArgs(args); err != nil {
			// Expected error path: flags.ErrHelp. Anything else (unknown
			// command, invalid flag) is swallowed so we can still render
			// the root help for the caller.
			if ferr, ok := err.(*flags.Error); !ok || ferr.Type != flags.ErrHelp {
				// Retry at root level so the caller gets something useful.
				rootParser := newParser(factory())
				_, _ = rootParser.ParseArgs([]string{"-h"})
				parser = rootParser
			}
		}

		var buf bytes.Buffer
		parser.WriteHelp(&buf)
		return buf.String(), nil
	}
}
