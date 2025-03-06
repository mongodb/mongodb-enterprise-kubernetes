#!/usr/bin/env bash

set -eou pipefail

script_name=$(readlink -f "${BASH_SOURCE[0]}")
script_dir=$(dirname "${script_name}")

source scripts/code_snippets/sample_test_runner.sh

pushd "${script_dir}"

prepare_snippets

run 0045_create_namespaces.sh
run 0046_create_image_pull_secrets.sh

run_for_output 0200_kubectl_mongodb_configure_multi_cluster.sh
run_for_output 0205_helm_configure_repo.sh
run_for_output 0210_helm_install_operator.sh
run_for_output 0211_check_operator_deployment.sh

popd
