#!/bin/bash

########################################################
############# TEST CONFIGURATION #######################
########################################################


TIERS="hermetic docker aws gcp ascloud"
export AEROLAB_TELEMETRY_DISABLE=1
export FEATURES_FILE=/Users/rglonek/features.conf
# export AEROLAB_TEST_SEQUENTIAL=1 # run tiers sequentially
# export AEROLAB_SKIP_CLEANUP=1 # debug: leave instances/volumes/firewalls behind after the run

# The ascloud tier (`aerolab cloud ...` against the real Aerospike Cloud API)
# needs API credentials, which aerolab reads straight from the environment.
# Export them here, or set them in the calling shell; the tier is dropped from
# the run when they are absent. AEROSPIKE_CLOUD_ENV=dev targets the dev control
# plane instead of production.
# export AEROSPIKE_CLOUD_KEY=...
# export AEROSPIKE_CLOUD_SECRET=...
export AEROSPIKE_CLOUD_KEY AEROSPIKE_CLOUD_SECRET AEROSPIKE_CLOUD_ENV

########################################################
############# TEST CODE STARTS HERE ####################
########################################################

# Logs land in tests/log/ and are committed alongside the code they exercised,
# so the repo always carries the results of the last full run.
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="${TESTS_DIR}/log"
mkdir -p "${LOG_DIR}"
cd "${TESTS_DIR}/.." || exit 1

tier_hermetic() {
	make test-nogen
}

tier_docker() {
	AEROLAB_TEST_CUSTOM_TMPDIR=/tmp/aerolab-docker-test \
	    AEROLAB_FEATURES_FILE=${FEATURES_FILE} \
		AEROLAB_E2E_OS_MATRIX=1 \
		AEROLAB_E2E_EXTENDED=1 \
		make test-docker-nogen
}

tier_aws() {
	AEROLAB_TEST_CUSTOM_TMPDIR=/tmp/aerolab-aws-test \
		AEROLAB_CLOUD=aws AWS_PROFILE=eks \
		AEROLAB_AWS_TEST_REGIONS=ca-central-1 \
		AEROLAB_FEATURES_FILE=${FEATURES_FILE} \
		AEROLAB_E2E_AWS_REGION=ca-central-1 \
		AEROLAB_E2E_MIGRATE=1 \
		AEROLAB_TEST_DNS_DOMAIN=test.aerolab.aerospike.com \
		AEROLAB_TEST_DNS_ZONE_ID=Z0351445OSHKM8RBC7BQ \
		make test-cloud-nogen
}

tier_gcp() {
	AEROLAB_TEST_CUSTOM_TMPDIR=/tmp/aerolab-gcp-test \
		AEROLAB_CLOUD=gcp \
		AEROLAB_GCP_NO_PUBLIC_IP=1 \
		AEROLAB_GCP_USE_IAP=1 \
		GCP_PROJECT=aerolab-test-project-2 \
		AEROLAB_GCP_TEST_REGIONS=us-central1 \
		AEROLAB_FEATURES_FILE=${FEATURES_FILE} \
		AEROLAB_TEST_DNS_DOMAIN=testgcp.aerolab.aerospike.com \
		AEROLAB_TEST_DNS_ZONE_ID=testaerolab \
		make test-cloud-nogen
}

# The `aerolab cloud ...` commands: the Aerospike Cloud API plus the AWS calls
# those commands make for VPC peering and S3 log access. us-west-2 rather than
# the aws tier's ca-central-1 so the two share the account without sharing a
# region -- the aws tier's backend suite deletes instances, volumes, firewalls
# and images in the regions it is pointed at.
tier_ascloud() {
	AEROLAB_TEST_CUSTOM_TMPDIR=/tmp/aerolab-ascloud-test \
	    AWS_PROFILE=eks \
		AEROLAB_ASCLOUD_AWS_PROFILE=eks \
		AEROLAB_ASCLOUD_REGION=us-west-2 \
		make test-aerospike-cloud-nogen
}

# run_tier writes one log per tier, bracketed by a timestamp and ending with the
# tier's exit code, and exits with that code so `wait` can pick it up.
run_tier() {
	local name=$1
	local log="${LOG_DIR}/${name}.log"
	date > "${log}"
	"tier_${name}" >> "${log}" 2>&1
	local ret=$?
	echo ${ret} >> "${log}"
	date >> "${log}"
	echo "$(date) ==================== ${name} finished, exit ${ret} ===================="
	return ${ret}
}

# go generate rewrites shared embed artifacts in place (it deletes and recreates
# expiry.linux.amd64.zip and pkg/webui/dist), so it runs once here instead of as
# a prerequisite of each make target: four concurrent copies would race on those
# files, and the npm build that dominates it is wasted work three times over.
echo "$(date) ==================== Generating embedded assets ===================="
date > "${LOG_DIR}/generate.log"
if ! make generate >> "${LOG_DIR}/generate.log" 2>&1; then
	date >> "${LOG_DIR}/generate.log"
	echo "$(date) ==================== generate FAILED, see ${LOG_DIR}/generate.log ===================="
	exit 1
fi
date >> "${LOG_DIR}/generate.log"

# Every test in the ascloud tier skips without the Aerospike Cloud API
# credentials, which would report the tier as a green run that provisioned and
# verified nothing. Drop it from the list and say so instead.
if [ -z "${AEROSPIKE_CLOUD_KEY}" ] || [ -z "${AEROSPIKE_CLOUD_SECRET}" ]; then
	KEPT=""
	for tier in ${TIERS}; do
		if [ "${tier}" = "ascloud" ]; then
			echo "$(date) ==================== Skipping ascloud tier: AEROSPIKE_CLOUD_KEY / AEROSPIKE_CLOUD_SECRET not set ===================="
			continue
		fi
		KEPT="${KEPT} ${tier}"
	done
	TIERS="${KEPT}"
fi

RETX=0
RESULTS=""

if [ -n "${AEROLAB_TEST_SEQUENTIAL}" ]; then
	for tier in ${TIERS}; do
		echo "$(date) ==================== Running ${tier} tests ===================="
		run_tier "${tier}"
		ret=$?
		RETX=$((RETX + ret))
		RESULTS="${RESULTS} ${tier}=${ret}"
	done
else
	# bash 3.2 (the macOS system bash) has no associative arrays, so the tier
	# name and its pid are carried together as "name:pid".
	jobs=""
	for tier in ${TIERS}; do
		echo "$(date) ==================== Starting ${tier} tests ===================="
		run_tier "${tier}" &
		jobs="${jobs} ${tier}:$!"
	done
	for job in ${jobs}; do
		wait "${job##*:}"
		ret=$?
		RETX=$((RETX + ret))
		RESULTS="${RESULTS} ${job%%:*}=${ret}"
	done
fi

echo "$(date) ==================== All tests completed ===================="
echo "$(date) ==================== Exit codes:${RESULTS} ===================="
echo "$(date) ==================== Logs saved to ${LOG_DIR}/{generate,hermetic,docker,aws,gcp,ascloud}.log ===================="
echo ${RETX}
exit ${RETX}
