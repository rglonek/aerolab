#!/bin/bash

# NOTE: everything tests the whole aerolab EXCEPT the aerospike cloud integrations; that's by design
# run as ./everything.sh 2>&1 |tee everything.log
#export AEROLAB_SKIP_CLEANUP=1            # debug: leave instances/volumes/firewalls behind after the run

# The DNS test in tests/backend needs a hosted zone you own; it skips unless
# these are set. ZONE_ID is the Route53 zone id on aws, the managed zone name on gcp.
#export AEROLAB_TEST_DNS_DOMAIN=aerospike.me
#export AEROLAB_TEST_DNS_ZONE_ID=Z08885863MUP8ENZ1K1Z7   # gcp: aerospikeme

set -e

export AEROLAB_TELEMETRY_DISABLE=1
FEATURES_FILE=/Users/rglonek/features.conf

echo "==================== Running hermetic unit and mock tests ===================="
make test

echo "==================== Running docker e2e tests ===================="
AEROLAB_FEATURES_FILE=${FEATURES_FILE} \
AEROLAB_E2E_OS_MATRIX=1 \
AEROLAB_E2E_EXTENDED=1 \
make test-docker

echo "==================== Running aws e2e tests ===================="
AEROLAB_TEST_CUSTOM_TMPDIR=/tmp/aerolab-aws-test \
AEROLAB_CLOUD=aws AWS_PROFILE=eks \
AEROLAB_AWS_TEST_REGIONS=us-east-1 \
AEROLAB_FEATURES_FILE=${FEATURES_FILE} \
AEROLAB_E2E_AWS_REGION=us-east-1 \
AEROLAB_E2E_MIGRATE=1 \
make test-cloud

echo "==================== Running gcp e2e tests ===================="
AEROLAB_TEST_CUSTOM_TMPDIR=/tmp/aerolab-gcp-test \
AEROLAB_CLOUD=gcp \
AEROLAB_GCP_NO_PUBLIC_IP=1 \
AEROLAB_GCP_USE_IAP=1 \
GCP_PROJECT=aerolab-test-project-2 \
AEROLAB_GCP_TEST_REGIONS=us-central1 \
AEROLAB_FEATURES_FILE=${FEATURES_FILE} \
make test-cloud
