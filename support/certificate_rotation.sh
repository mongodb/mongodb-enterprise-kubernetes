#!/usr/bin/env bash

#
# certificate_rotation.sh
#
# WARNING: This script is provided as a guide-only and it is not meant
# to be used in production environment.
#
# Use this script as a guide on how to rotate TLS certificates on a
# MongoDB Resource. During this process there will be no downtime on
# the Mdb resource.
#

#
# shellcheck disable=SC2119
# shellcheck disable=SC2039
#

usage() {
    local script_name
    script_name=$(basename "${0}")
    echo "Usage:"
    echo "${script_name} <namespace> <mdb_resource_name>"
}

if [ -z "${2}" ]; then
    usage
    exit 1
fi

namespace="${1}"
mdb_resource_name="${2}"
mdb_resource_type=$(kubectl -n "${namespace}" get "mdb/${mdb_resource_name}" -o jsonpath='{.spec.type}')
mdb_resource_members=$(kubectl -n "${namespace}" get "mdb/${mdb_resource_name}" -o jsonpath='{.spec.members}')
mdb_resource_members=$(("${mdb_resource_members}" - 1))

if [[ ${mdb_resource_type} != "ReplicaSet" ]]; then
    echo "Only Replica Set TLS certificates are supported as of now."
    exit 1
fi

echo "Removing existing CSRs if they still exist."
for i in $(seq 0 ${mdb_resource_members}); do
    kubectl delete "csr/${mdb_resource_name}-${i}.${namespace}" || true
done

echo "Removing the 'Secret' object holding the current certificates and private keys."
kubectl -n "${namespace}" delete "secret/${mdb_resource_name}-cert"

timestamp=$(date --rfc-3339=ns)
echo "Triggering a reconciliation for the Operator to notice the missing certs."
kubectl -n "${namespace}" patch "mdb/${mdb_resource_name}" --type='json' \
    -p='[{"op": "add", "path": "/metadata/annotations/timestamp", "value": "'"${timestamp}"'"}]'

echo "Wait until the operator recreates the CSRs."
while true; do
    all_created=0
    for i in $(seq 0 "${mdb_resource_members}"); do
        if ! kubectl get "csr/${mdb_resource_name}-${i}.${namespace}" -o name > /dev/null ; then
            all_created=1
        fi
    done
    if [[ ${all_created} != 0 ]]; then
        sleep 10
    else
        break
    fi
done

echo "CSRs have been generated. Approving certificates."
for i in $(seq 0 ${mdb_resource_members}); do
    kubectl certificate approve "${mdb_resource_name}-${i}.${namespace}"
done

echo "A this point, the operator should take the new certificates and generate the Secret."
while ! kubectl -n "${namespace}" get "secret/${mdb_resource_name}-cert" &> /dev/null; do
    printf "."
    sleep 10
done

echo "Secret with certificates has been created, proceeding with a rolling restart of the Mdb resource"
kubectl -n "${namespace}" rollout restart sts "${mdb_resource_name}"

echo "The Mdb resource is being restarted now, it should take a few minutes to reach Running state again."
