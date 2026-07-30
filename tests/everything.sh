#!/bin/bash

# NOTE: everything tests the whole aerolab EXCEPT the aerospike cloud integrations; that's by design
# run as ./everything.sh 2>&1 |tee everything.log
#export AEROLAB_SKIP_CLEANUP=1            # debug: leave instances/volumes/firewalls behind after the run

set -e

echo "==================== Running hermetic unit and mock tests ===================="
make test

echo "==================== Running docker e2e tests ===================="
AEROLAB_TEST_CUSTOM_TMPDIR=/Users/rglonek/features.conf \
AEROLAB_FEATURES_FILE=/path/to/features.conf \
AEROLAB_E2E_OS_MATRIX=1 \
AEROLAB_E2E_EXTENDED=1 \
make test-docker

echo "==================== Running aws e2e tests ===================="
AEROLAB_TEST_CUSTOM_TMPDIR=/tmp/aerolab-aws-test \
AEROLAB_CLOUD=aws AWS_PROFILE=eks \
AEROLAB_FEATURES_FILE=/Users/rglonek/features.conf \
AEROLAB_E2E_AWS_REGION=us-east-1 \
AEROLAB_E2E_MIGRATE=1 \
make test-cloud

echo "==================== Running gcp e2e tests ===================="
AEROLAB_TEST_CUSTOM_TMPDIR=/Users/rglonek/features.conf \
AEROLAB_CLOUD=gcp \
AEROLAB_GCP_NO_PUBLIC_IP=1 \
AEROLAB_GCP_USE_IAP=1 \
GCP_PROJECT=aerolab-test-project-2 \
AEROLAB_GCP_TEST_REGIONS=us-central1 \
make test-cloud
