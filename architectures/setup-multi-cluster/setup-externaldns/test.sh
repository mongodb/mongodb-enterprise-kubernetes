#!/usr/bin/env bash

set -eou pipefail

script_name=$(readlink -f "${BASH_SOURCE[0]}")
script_dir=$(dirname "${script_name}")

source scripts/code_snippets/sample_test_runner.sh

pushd "${script_dir}"

prepare_snippets

run 0100_create_gke_sa.sh
# need to wait as the SA is not immediately available
sleep 10
run 0120_add_role_to_sa.sh
run 0130_create_sa_key.sh
run 0140_create_namespaces.sh
run 0150_create_sa_secrets.sh
run 0200_install_externaldns.sh
run 0300_setup_dns_zone.sh

popd
