#!/usr/bin/env bash
#
# test-all.sh — set the test variables and run every tier in order.
# Edit the values below to match your environment, then run: ./test-all.sh
#
# Run from the src/ directory (same place as the Makefile).

set -e

# --- always safe for tests ---
export AEROLAB_TEST=1
export AEROLAB_TELEMETRY_DISABLE=1

# --- Docker tier: needs a running Docker daemon ---
# Path to your Aerospike features file (Enterprise images need it).
export AEROLAB_FEATURES_FILE="$HOME/features.conf"
# Run the extra Docker e2e suites too.
export AEROLAB_E2E_OS_MATRIX=1     # deploy every supported distro
export AEROLAB_E2E_EXTENDED=1      # TLS / XDR / data / net / clients

# --- Cloud tier: needs real AWS or GCP credentials ---
export AEROLAB_CLOUD=aws           # aws or gcp
export AWS_PROFILE=my-aws-profile  # your AWS profile (for aws)
export GCP_PROJECT=my-gcp-project  # your GCP project (for gcp)
export AEROLAB_AWS_TEST_REGIONS=us-east-1
export AEROLAB_GCP_TEST_REGIONS=us-central1
# Cloud e2e (Aerospike Cloud databases/secrets):
export AEROLAB_E2E_CLOUD=1
export AEROLAB_E2E_AWS_REGION=us-east-1
export AEROLAB_E2E_VPC_ID=vpc-xxxxxxxx   # VPC to create the test database in
export AEROLAB_E2E_MIGRATE=1
export AEROLAB_E2E_SSH_KEY_PATH="$HOME/aerolab-keys"

# --- run every tier in order ---
echo "==> Unit + mock tests"
make test

echo "==> Docker integration + e2e tests"
make test-docker

echo "==> Cloud integration + e2e tests"
make test-cloud

echo "==> All tests done"
