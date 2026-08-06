#!/bin/bash
# shellcheck disable=SC1083
DEBUG="{{.Debug}}"

if [ "$DEBUG" == "true" ]; then
    set -x
fi

# set -u # if variable is not set, error
# set -o pipefail # die if any command in a pipeline fails, not just last one
#
# Note: this script deliberately does not use `set -e`. Bash exempts commands in
# an && or || list from errexit, so a failing `apt-get update` in an
# `update && install` chain would silently skip the install and let the script
# continue. Every command that must succeed is checked explicitly instead.

# Retry helper function: tries command once, sleeps 1s, then retries once on failure
retry_cmd() {
    "$@" || { sleep 1; "$@"; }
}

# Detect the package manager once, so the rest of the script can branch on the
# distribution family instead of re-probing for each step.
PKG_MGR=""
if command -v apt-get &> /dev/null; then
    PKG_MGR="apt"
elif command -v dnf &> /dev/null; then
    PKG_MGR="dnf"
elif command -v yum &> /dev/null; then
    PKG_MGR="yum"
fi

# Repositories for end-of-life distributions get moved off the primary mirrors.
# This must run before any package operation, not just before installing
# dependencies: installing an already-downloaded .rpm or .deb still resolves that
# package's own dependencies through the configured repositories.
patch_eol_repositories() {
    # CentOS Stream 8: mirrorlist.centos.org no longer resolves, the content was
    # moved to vault.centos.org.
    if [ -f /etc/redhat-release ] && grep -q "CentOS Stream release 8" /etc/redhat-release; then
        echo "Patching yum repositories for centos:stream8"
        sed -i 's/mirror.centos.org/vault.centos.org/g' /etc/yum.repos.d/*.repo
        sed -i 's/^#.*baseurl=http/baseurl=http/g' /etc/yum.repos.d/*.repo
        sed -i 's/^mirrorlist=http/#mirrorlist=http/g' /etc/yum.repos.d/*.repo
    fi
    # Debian 10 and older: deb.debian.org no longer carries these suites, and the
    # archived Release files are past their Valid-Until date. There is no
    # -updates suite on the archive, so drop it.
    if [ "$PKG_MGR" == "apt" ] && [ -f /etc/debian_version ] && [ -f /etc/apt/sources.list ]; then
        case "$(cat /etc/debian_version)" in
            8.*|9.*|10.*)
                echo "Patching apt repositories for end-of-life debian"
                sed -i '/-updates/d' /etc/apt/sources.list
                sed -i 's|https\?://deb.debian.org|http://archive.debian.org|g' /etc/apt/sources.list
                sed -i 's|https\?://security.debian.org|http://archive.debian.org|g' /etc/apt/sources.list
                echo 'Acquire::Check-Valid-Until "false";' > /etc/apt/apt.conf.d/99-aerolab-archive
                ;;
        esac
    fi
}
patch_eol_repositories

# if apt - disable unattended-upgrades, and handle conflicts for configuration files
if [ "$PKG_MGR" == "apt" ]; then
# unattended upgrades
export DEBIAN_FRONTEND=noninteractive
apt-get update || true
systemctl stop unattended-upgrades || true
pkill --signal SIGKILL unattended-upgrades || true
systemctl disable unattended-upgrades || true
apt-get -y -f install || true
apt-get -y purge unattended-upgrades || true
# restart if we have to
sed -i.bak "/#\$nrconf{restart} = .*/s/.*/\$nrconf{restart} = 'a';/" /etc/needrestart/needrestart.conf || true
# conflict handling for configuration files
cat <<'EOF' > /etc/apt/apt.conf.d/local || true
Dpkg::Options {
	"--force-confdef";
	"--force-confold";
}
EOF
cat <<'EOF' > /etc/dpkg/dpkg.cfg.d/local || true
force-confdef
force-confold
EOF
fi

# if rpm-based - disable sshd-keygen cloud init
if [ "$PKG_MGR" == "dnf" ] || [ "$PKG_MGR" == "yum" ]; then
    rm -f /etc/systemd/system/sshd-keygen\@.service.d/disable-sshd-keygen-if-cloud-init-active.conf
    if command -v systemctl &> /dev/null; then
        systemctl daemon-reload || true
    fi
fi

# install_packages installs every package passed to it, treating failure as
# fatal. Callers that can tolerate a missing package install one at a time.
install_packages() {
    case "$PKG_MGR" in
        apt)
            if [ ! -e /etc/localtime ]; then ln -fs /usr/share/zoneinfo/UTC /etc/localtime; fi
            retry_cmd apt-get update || { echo "Failed to update apt package lists"; exit 1; }
            DEBIAN_FRONTEND=noninteractive retry_cmd apt-get install -y "$@" || { echo "Failed to install: $*"; exit 1; }
            ;;
        dnf)
            retry_cmd dnf install -y "$@" || { echo "Failed to install: $*"; exit 1; }
            ;;
        yum)
            retry_cmd yum install -y "$@" || { echo "Failed to install: $*"; exit 1; }
            ;;
        *)
            echo "Could not find package manager to install dependencies"
            exit 1
            ;;
    esac
}

# check dependencies
# names of packages to install the dependencies, so if dependency 0 "curl" is missing, it will install package 0 "curl".
# package names differ between distribution families (openssh-client vs
# openssh-clients, for instance), so both are emitted and the right one picked.
DEPS=({{ range .Required.Dependencies }}"{{ .Command }}" {{ end }})
PACKAGES_APT=({{ range .Required.Dependencies }}"{{ .AptPackage }}" {{ end }})
PACKAGES_RPM=({{ range .Required.Dependencies }}"{{ .RpmPackage }}" {{ end }})
TO_INSTALL=({{ range .Required.Packages }}"{{ . }}" {{ end }})
if [ "$PKG_MGR" == "apt" ]; then
    PACKAGES=("${PACKAGES_APT[@]}")
else
    PACKAGES=("${PACKAGES_RPM[@]}")
fi
for i in "${!DEPS[@]}"; do
    dep="${DEPS[$i]}"
    if ! command -v "$dep" &> /dev/null; then
        echo "Could not find $dep, adding to install list"
        TO_INSTALL+=("${PACKAGES[$i]}")
    fi
done

DEPS_OPTIONAL=({{ range .Optional.Dependencies }}"{{ .Command }}" {{ end }})
PACKAGES_OPTIONAL_APT=({{ range .Optional.Dependencies }}"{{ .AptPackage }}" {{ end }})
PACKAGES_OPTIONAL_RPM=({{ range .Optional.Dependencies }}"{{ .RpmPackage }}" {{ end }})
TO_INSTALL_OPTIONAL=({{ range .Optional.Packages }}"{{ . }}" {{ end }})
if [ "$PKG_MGR" == "apt" ]; then
    PACKAGES_OPTIONAL=("${PACKAGES_OPTIONAL_APT[@]}")
else
    PACKAGES_OPTIONAL=("${PACKAGES_OPTIONAL_RPM[@]}")
fi
for i in "${!DEPS_OPTIONAL[@]}"; do
    dep="${DEPS_OPTIONAL[$i]}"
    if ! command -v "$dep" &> /dev/null; then
        echo "Could not find $dep, adding to install list"
        TO_INSTALL_OPTIONAL+=("${PACKAGES_OPTIONAL[$i]}")
    fi
done

# install dependencies
if [ ${#TO_INSTALL[@]} -gt 0 ]; then
    echo "Installing dependencies: ${TO_INSTALL[*]}"
    install_packages "${TO_INSTALL[@]}"
fi

# install optional dependencies
if [ ${#TO_INSTALL_OPTIONAL[@]} -gt 0 ]; then
    echo "Installing dependencies: ${TO_INSTALL_OPTIONAL[*]}"
    if [ "$PKG_MGR" == "apt" ]; then
        if [ ! -e /etc/localtime ]; then ln -fs /usr/share/zoneinfo/UTC /etc/localtime; fi
        apt-get update || true
        for pkg in "${TO_INSTALL_OPTIONAL[@]}"; do
            DEBIAN_FRONTEND=noninteractive apt-get install -y "$pkg" || true
        done
    elif [ "$PKG_MGR" == "dnf" ] || [ "$PKG_MGR" == "yum" ]; then
        for pkg in "${TO_INSTALL_OPTIONAL[@]}"; do
            "$PKG_MGR" install -y "$pkg" || true
        done
    else
        echo "Could not find package manager to install optional dependencies"
    fi
fi
