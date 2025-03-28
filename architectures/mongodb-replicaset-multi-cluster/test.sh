#!/usr/bin/env bash

set -eou pipefail

script_name=$(readlink -f "${BASH_SOURCE[0]}")
script_dir=$(dirname "${script_name}")

source scripts/code_snippets/sample_test_runner.sh

pushd "${script_dir}"

prepare_snippets

run 1050_generate_certs.sh
run 1100_mongodb_replicaset_multi_cluster.sh
run 1110_mongodb_replicaset_multi_cluster_wait_for_running_state.sh

run 1200_create_mongodb_user.sh
sleep 10
run_for_output 1210_verify_mongosh_connection.sh

popd
