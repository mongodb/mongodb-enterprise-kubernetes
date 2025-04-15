#!/usr/bin/env bash

set -eou pipefail

script_name=$(readlink -f "${BASH_SOURCE[0]}")
script_dir=$(dirname "${script_name}")

source scripts/code_snippets/sample_test_runner.sh

pushd "${script_dir}"

prepare_snippets

run 9000_delete_sa.sh
run 9050_delete_namespace.sh
run 9100_delete_dns_zone.sh

popd
