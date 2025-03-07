#!/usr/bin/env bash

set -eou pipefail

script_name=$(readlink -f "${BASH_SOURCE[0]}")
script_dir=$(dirname "${script_name}")

source scripts/code_snippets/sample_test_runner.sh

pushd "${script_dir}"

prepare_snippets

run 0005_gcloud_set_current_project.sh
run 0010_create_gke_cluster_0.sh &
run 0010_create_gke_cluster_1.sh &
run 0010_create_gke_cluster_2.sh &
wait

run 0020_get_gke_credentials.sh
run_for_output 0030_verify_access_to_clusters.sh

popd
