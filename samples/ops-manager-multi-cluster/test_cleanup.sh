#!/usr/bin/env bash

# This script only cleans up local directory to prepare to a fresh run. It's not cleaning up any deployed resources/clusters.

set -eou pipefail

source env_variables.sh
source ../../scripts/sample_test_runner.sh

run_cleanup "test.sh"
rm -rf istio*
rm -rf certs
