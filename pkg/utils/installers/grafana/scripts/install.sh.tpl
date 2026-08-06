#!/bin/bash
VERSION="{{.Version}}"
ENABLE_GRAFANA="{{.EnableGrafana}}"
START_GRAFANA="{{.StartGrafana}}"

# Retry helper function: tries command once, sleeps 1s, then retries once on failure
retry_cmd() {
    "$@" || { sleep 1; "$@"; }
}

# Determine package manager
if command -v apt-get &> /dev/null; then
    # Debian/Ubuntu
    if [ ! -f /etc/localtime ]; then
        ln -fs /usr/share/zoneinfo/America/Los_Angeles /etc/localtime
    fi
    set -e
    retry_cmd apt-get update
    DEBIAN_FRONTEND=noninteractive retry_cmd apt-get install -y apt-transport-https software-properties-common wget
    retry_cmd wget -q -O /usr/share/keyrings/grafana.key https://apt.grafana.com/gpg.key
    echo "deb [signed-by=/usr/share/keyrings/grafana.key] https://apt.grafana.com stable main" | tee /etc/apt/sources.list.d/grafana.list
    retry_cmd apt-get update
    set +e
    if [ -z "$VERSION" ]; then
        retry_cmd apt-get install -y grafana || exit 1
    else
        retry_cmd apt-get install -y grafana="$VERSION" || exit 1
    fi

elif command -v yum &> /dev/null; then
    # RHEL/CentOS
    if [ -f /etc/redhat-release ] && grep -q "CentOS Stream release 8" /etc/redhat-release; then
        echo "Patching yum for centos:stream8"
        sed -i 's/mirror.centos.org/vault.centos.org/g' /etc/yum.repos.d/*.repo
        sed -i 's/^#.*baseurl=http/baseurl=http/g' /etc/yum.repos.d/*.repo
        sed -i 's/^mirrorlist=http/#mirrorlist=http/g' /etc/yum.repos.d/*.repo
    fi
    cat <<EOF > /etc/yum.repos.d/grafana.repo || exit 1
[grafana]
name=grafana
baseurl=https://rpm.grafana.com
repo_gpgcheck=1
enabled=1
gpgcheck=1
gpgkey=https://rpm.grafana.com/gpg.key
sslverify=1
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
EOF

    # rpm.grafana.com intermittently serves a repomd.xml that does not match its
    # detached signature, which yum reports as "repomd.xml GPG signature
    # verification error: Bad GPG signature" and then caches, so retrying within
    # a second just re-reads the same bad copy. Drop the cached metadata between
    # attempts and back off, rather than turning off repo_gpgcheck.
    yum_install_grafana() {
        local attempt
        for attempt in 1 2 3; do
            if yum install -y "$@"; then
                return 0
            fi
            echo "grafana install attempt $attempt failed, clearing repo metadata and retrying"
            yum clean metadata --disablerepo='*' --enablerepo=grafana || yum clean all || true
            sleep $((attempt * 5))
        done
        return 1
    }

    if [ -z "$VERSION" ]; then
        yum_install_grafana grafana || exit 1
    else
        yum_install_grafana grafana-"$VERSION" || exit 1
    fi

else
    echo "Neither apt-get nor yum found. This script requires one of these package managers."
    exit 1
fi

if [ "$ENABLE_GRAFANA" == "true" ]; then
    systemctl daemon-reload || exit 1
    systemctl enable grafana-server || exit 1
fi

if [ "$START_GRAFANA" == "true" ]; then
    systemctl daemon-reload || exit 1
    systemctl start grafana-server || exit 1
fi

echo "Grafana installation completed"
