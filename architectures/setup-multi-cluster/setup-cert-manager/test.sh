#!/usr/bin/env bash

set -eou pipefail

script_name=$(readlink -f "${BASH_SOURCE[0]}")
script_dir=$(dirname "${script_name}")

source scripts/code_snippets/sample_test_runner.sh

pushd "${script_dir}"

prepare_snippets

run_for_output 0215_helm_configure_repo.sh
run_for_output 0216_helm_install_cert_manager.sh
run 0220_create_issuer.sh
run_for_output 0221_verify_issuer.sh
run 0225_create_ca_configmap.sh

popd
