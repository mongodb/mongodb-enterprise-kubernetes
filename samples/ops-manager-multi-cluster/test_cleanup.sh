#!/usr/bin/env bash

set -eou pipefail

source env_variables.sh
source ../../scripts/sample_test_runner.sh

run_cleanup "test.sh"
