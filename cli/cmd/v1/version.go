package cmd

import (
	_ "embed"
	"strings"
)

// The embed_*.txt files are committed with default contents ("unknown"
// commit, VERSION.md branch, "-unofficial" edition) so this package is
// importable as a library without a generate step. Real values are stamped
// by the Makefile `prep`/`official`/`prerelease` targets at build time.
// (The expiry function binary is compiled via go:generate in
// pkg/backend/backends/expiry.go.)

//go:embed embed_commit.txt
var vCommit string

//go:embed embed_branch.txt
var vBranch string

//go:embed embed_tail.txt
var vEdition string

func GetAerolabVersion() (branch, commit, edition, friendlyString string) {
	branch = strings.Trim(vBranch, "\t\r\n ")
	commit = strings.Trim(vCommit, "\t\r\n ")
	edition = strings.Trim(vEdition, "\t\r\n ")
	friendlyString = "v" + branch + "-" + commit + edition
	return
}
