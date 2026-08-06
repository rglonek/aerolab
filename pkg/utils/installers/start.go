package installers

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed dependencies.sh.tpl
var dependenciesScriptTemplate []byte

type Dependency struct {
	Command string `json:"command"` // command to check for
	Package string `json:"package"` // package to install if command not found
	// PackageRPM overrides Package on rpm-based distributions, for the cases
	// where the two families name the same package differently
	// (openssh-client on debian, openssh-clients on rhel). Empty means Package
	// is correct for both.
	PackageRPM string `json:"packageRpm,omitempty"`
}

// AptPackage is the package providing Command on apt-based distributions. It is
// referenced from the dependency install script template.
func (d Dependency) AptPackage() string {
	return d.Package
}

// RpmPackage is the package providing Command on rpm-based distributions. It is
// referenced from the dependency install script template.
func (d Dependency) RpmPackage() string {
	if d.PackageRPM != "" {
		return d.PackageRPM
	}
	return d.Package
}

type Installs struct {
	Dependencies []Dependency `json:"dependencies"` // commands to check for and packages to install if command not found
	Packages     []string     `json:"packages"`     // packages to install always
}

type Software struct {
	Debug    bool     `json:"debug"`    // if true, print debug information (set -x)
	Required Installs `json:"required"` // required packages (fail if cannot install)
	Optional Installs `json:"optional"` // optional packages (try and install one at a time and continue if error)
}

func GetInstallScript(software Software, tailScript []byte) ([]byte, error) {
	script, err := processTemplate(dependenciesScriptTemplate, software)
	if err != nil {
		return nil, err
	}

	return append(script, tailScript...), nil
}

func processTemplate(scriptFile []byte, data any) ([]byte, error) {
	tmpl, err := template.New("script").Parse(string(scriptFile))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
