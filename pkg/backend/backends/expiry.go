package backends

import (
	_ "embed"
	"errors"
)

//go:generate bash -c "cd ../../expiry && bash compile.sh"

//go:embed expiry.version.txt
var ExpiryVersion string

// ErrNoExpiryBinary is returned by expiry deployment paths when this build
// does not carry the embedded expiry function binary. The binary is only
// embedded when building with -tags=embedexpiry (the Makefile does this);
// plain `go build`/`go get` builds get a stub so the module is importable
// without running `go generate` first.
var ErrNoExpiryBinary = errors.New("this build does not include the embedded expiry binary; build aerolab via make (or with -tags=embedexpiry after running `go generate ./...`) to deploy expiry systems")

type ExpiryList struct {
	ExpirySystems []*ExpirySystem `yaml:"expirySystems" json:"expirySystems"`
}

type ExpirySystem struct {
	BackendType         BackendType `yaml:"backendType" json:"backendType"`
	Zone                string      `yaml:"zone" json:"zone"`
	Version             string      `yaml:"version" json:"version"`
	InstallationSuccess bool        `yaml:"installationSuccess" json:"installationSuccess"`
	FrequencyMinutes    int         `yaml:"frequencyMinutes" json:"frequencyMinutes"`
	BackendSpecific     any         `yaml:"backendSpecific" json:"backendSpecific"`
}

func (b *backend) ExpiryList() (*ExpiryList, error) {
	ret := &ExpiryList{}
	for c := range b.enabledBackends {
		expirySystems, err := b.enabledBackends[c].ExpiryList()
		if err != nil {
			return ret, err
		}
		ret.ExpirySystems = append(ret.ExpirySystems, expirySystems...)
	}
	return ret, nil
}

// install the expiry system for the given backend type
// intervalMinutes is the interval in minutes between expiry checks; it is advised to set this to 15 minutes or more, 0=default 15 minutes
// logLevel is the log level for the expiry system; 0=default 3 (info)
// expireEksctl is true if eksctl should be expired; only applies to AWS
// cleanupDNS is true if DNS should be cleaned up
// force is true if the expiry system should be installed even if it already exists
// onUpdateKeepOriginalSettings is true if the original settings should be kept on update
func (b *backend) ExpiryInstall(backendType BackendType, intervalMinutes int, logLevel int, expireEksctl bool, cleanupDNS bool, force bool, onUpdateKeepOriginalSettings bool, zones ...string) error {
	if intervalMinutes == 0 {
		intervalMinutes = 15
	}
	if logLevel == 0 {
		logLevel = 3
	}
	return b.enabledBackends[backendType].ExpiryInstall(intervalMinutes, logLevel, expireEksctl, cleanupDNS, force, onUpdateKeepOriginalSettings, zones...)
}

// remove the expiry system for the given backend type
func (b *backend) ExpiryRemove(backendType BackendType, zones ...string) error {
	return b.enabledBackends[backendType].ExpiryRemove(zones...)
}

// change the frequency of the expiry system for the given backend type
// intervalMinutes is the interval in minutes between expiry checks; it is advised to set this to 15 minutes or more, 0=default 15 minutes
func (b *backend) ExpiryChangeFrequency(backendType BackendType, intervalMinutes int, zones ...string) error {
	if intervalMinutes == 0 {
		intervalMinutes = 15
	}
	return b.enabledBackends[backendType].ExpiryChangeFrequency(intervalMinutes, zones...)
}

// change the configuration of the expiry system for the given backend type
// logLevel is the log level for the expiry system; 0=default 3 (info)
// expireEksctl is true if eksctl should be expired; only applies to AWS
// cleanupDNS is true if DNS should be cleaned up
func (b *backend) ExpiryChangeConfiguration(backendType BackendType, logLevel int, expireEksctl bool, cleanupDNS bool, zones ...string) error {
	if logLevel == 0 {
		logLevel = 3
	}
	return b.enabledBackends[backendType].ExpiryChangeConfiguration(logLevel, expireEksctl, cleanupDNS, zones...)
}

// ExpiryV7Check checks if the v7 expiry system is still installed for the given backend type.
// Returns true if v7 expiry system is detected, along with a list of regions where it was found.
func (b *backend) ExpiryV7Check(backendType BackendType) (bool, []string, error) {
	return b.enabledBackends[backendType].ExpiryV7Check()
}
