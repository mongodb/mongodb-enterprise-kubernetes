#!/usr/bin/env bash

set -eou pipefail

log_file="$0.run.log"
snippets_src_dir="code_snippets"
snippets_run_dir=".generated"

DEBUG=${DEBUG:-"false"}

function snippets_list() {
  src_dir=$1
  # shellcheck disable=SC2012
  ls -1 "${src_dir}" | sort -t '_' -k1,1n -k2,2
}

function run_cleanup() {
  script_file=$1
  rm -rf "${snippets_run_dir}" 2>/dev/null || true
  rm -rf "log" 2>/dev/null || true
  rm -rf "${script_file}.run.log" 2>/dev/null || true
}

function prepare_snippets() {
  echo "Generating code snippets in ${snippets_run_dir}..."

  touch "${log_file}"
  mkdir log 2>/dev/null || true
  mkdir output 2>/dev/null || true

  rm -rf "${snippets_run_dir}" 2>/dev/null || true
  mkdir "${snippets_run_dir}" 2>/dev/null || true

  file_list=$(snippets_list "${snippets_src_dir}")
  while IFS= read -r file_name; do
    file_path="${snippets_run_dir}/${file_name}"
    (
      echo "# This file is generated automatically from ${file_path}"
      echo "# DO NOT EDIT"
      echo "function ${file_name%.sh}() {"
      cat "${snippets_src_dir}/${file_name}"
      echo "}"
    ) > "${file_path}"
  done <<< "${file_list}"
}

function run() {
  # shellcheck disable=SC1090
  source "${snippets_run_dir}/$1"
  cmd=${1%.sh}

  if grep -q "^${cmd}$" "${log_file}"; then
    echo "Skipping ${cmd} as it is already executed."
    return 0
  fi

  echo "$(date +"%Y-%m-%d %H:%M:%S") Executing ${cmd}"

  stdout_file="log/${cmd}.stdout.log"
  stderr_file="log/${cmd}.stderr.log"
  set +e
  (set -e; set -x; "${cmd}" >"${stdout_file}" 2>"${stderr_file}")
  ret=$?
  set -e
  if [[ ${ret} == 0 ]]; then
    echo "${cmd}" >> "${log_file}"
  else
    echo "Error running: ${cmd}"
  fi

  if [[ ${DEBUG} == "true" || ${ret} != 0 ]]; then
    cat "${stdout_file}"
    cat "${stderr_file}"
  fi

  return ${ret}
}

function run_for_output() {
  # shellcheck disable=SC1090
  source "${snippets_run_dir}/$1"
  cmd=${1%.sh}

  if grep -q "^${cmd}$" "${log_file}"; then
    echo "Skipping ${cmd} as it is already executed."
    return 0
  fi

  echo "$(date +"%Y-%m-%d %H:%M:%S") Executing ${cmd}"
  stdout_file="log/${cmd}.stdout.log"
  stderr_file="log/${cmd}.stderr.log"
  set +e
  (set -e; set -x; "${cmd}" >"${stdout_file}" 2>"${stderr_file}")
  ret=$?
  set -e
  if [[ ${ret} == 0 ]]; then
    tee "output/${cmd}.out" < "${stdout_file}"
  else
    echo "Error running: ${cmd}"
  fi

  if [[ ${ret} == 0 ]]; then
    echo "${cmd}" >> "${log_file}"
  fi

  if [[ ${DEBUG} == "true" || ${ret} != 0 ]]; then
      cat "${stdout_file}"
      cat "${stderr_file}"
  fi

  return ${ret}
}
