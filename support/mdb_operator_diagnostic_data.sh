#!/usr/bin/env bash

set -Eeou pipefail

#
# mdb_operator_diagnostic_data.sh
#
# Use this script to gather data about your MongoDB Enterprise Kubernetes Operator
# and the MongoDB Resources deployed with it.
#

#
# shellcheck disable=SC2119
# shellcheck disable=SC2039
#

usage() {
    local script_name
    script_name=$(basename "${0}")
    echo "------------------------------------------------------------------------------"
    echo "Usage:"
    echo "${script_name} <namespace> <mdb/om_resource_name> [<operator_namespace>] [<operator_name>] [--om] [--private]"
    echo "------------------------------------------------------------------------------"
    echo "#Scenario 01: Collecting MongoDB Logs(Operator in same namespace as MongoDB):"
    echo "------------------------------------------------------------------------------"
    echo "Example: Operator_Namespace: mongodb, Deployment_Namespace: mongodb, Deployment_Name: myreplicaset"
    echo "Usage: ${script_name} mongodb myreplicaset"
    echo "For OpenShift: ${script_name} mongodb myreplicaset mongodb enterprise-operator"
    echo "------------------------------------------------------------------------------"
    echo "#Scenario 02: Collecting MongoDB Logs(Operator in different namespace as MongoDB):"
    echo "------------------------------------------------------------------------------"
    echo "Example: Operator_Namespace: mdboperator, Deployment_Namespace: mongodb, Deployment_Name: myreplicaset"
    echo "Usage: ${script_name} mongodb myreplicaset mdboperator"
    echo "For OpenShift: ${script_name} mongodb myreplicaset mdboperator enterprise-operator"
    echo "------------------------------------------------------------------------------"
    echo "#Scenario 03: Collecting Ops Manager Logs(Operator in same namespace as Ops Manager):"
    echo "------------------------------------------------------------------------------"
    echo "Example: Operator_Namespace: mongodb, Deployment_Namespace: mongodb, Deployment_Name: ops-manager"
    echo "Usage: ${script_name} mongodb ops-manager --om"
    echo "For OpenShift: ${script_name} mongodb ops-manager mongodb enterprise-operator --om"
    echo "------------------------------------------------------------------------------"
    echo "#Scenario 04: Collecting Ops Manager Logs(Operator in different namespace as Ops Manager):"
    echo "Example: Operator_Namespace: mdboperator, Deployment_Namespace: mongodb, Deployment_Name: ops-manager"
    echo "Usage: ${script_name} mongodb ops-manager mdboperator --om"
    echo "For OpenShift: ${script_name} mongodb ops-manager mdboperator enterprise-operator --om"
    echo "------------------------------------------------------------------------------"
}

contains() {
    local e match=$1
    shift
    for e; do [[ "$e" == "$match" ]] && return 0; done
    return 1
}

if [ $# -lt 2 ]; then
    usage >&2
    exit 1
fi

namespace="${1}"
mdb_resource="${2}"
mdb_container_name="mongodb-enterprise-database"

collect_om=0
contains "--om" "$@" && collect_om=1

if [ ${collect_om} == 1 ] 
then
    if [[ $3 == "--om" ]]
    then
        operator_namespace="${1}"
        operator_name="mongodb-enterprise-operator"
        om_resource_name="${2}"
    elif [[ $4 == "--om" ]]
    then
        operator_namespace="${3}"
        operator_name="mongodb-enterprise-operator"
        om_resource_name="${2}"
    elif [[ $5 == "--om" ]]
    then
        operator_namespace="${3}"
        operator_name="$4"
        om_resource_name="${2}"
    fi
else
    operator_name="${4:-mongodb-enterprise-operator}"
    operator_namespace="${3:-$1}"
fi

current_date="$(date +%Y-%m-%d_%H_%M)"

private_mode=1
contains "--private" "$@" && private_mode=0

log_dir="logs_${current_date}"
mkdir -p "${log_dir}" &> /dev/null



if ! kubectl get "namespace/${namespace}" &> /dev/null; then
    echo "Error fetching namespace. Make sure name ${namespace} for Namespace is correct."
    exit 1
fi

if [ ${collect_om} == 0 ]
then
    if ! kubectl -n "${namespace}" get "mdb/${mdb_resource}" &> /dev/null; then
        echo "Error fetching the MongoDB resource. Make sure the ${mdb_resource} is correct."
        exit 1
    fi
fi

if ! kubectl -n "${operator_namespace}" get "deployment/${operator_name}" &> /dev/null; then
    echo "Error fetching the MongoDB Operator Deployment. Make sure the deployment/${operator_name} exist and it is running."
    exit 1
fi

if ! kubectl -n "${namespace}" get om -o wide &> /dev/null; then
    echo "Error fetching the MongoDB OpsManager Resource."
fi


if [ ${private_mode} == 0 ]; then
    echo "+ Running on private mode. Make sure you don't share the results of this run outside your organization."
fi

echo "++ Versions"
mdb_operator_pod=$(kubectl -n "${operator_namespace}" get pods -l "app.kubernetes.io/component=controller" -o name | cut -d'/' -f 2)
echo "+ Operator Pod: pod/${mdb_operator_pod}"

mdb_operator_filename="operator.yaml"
echo "+ Saving Operator Deployment into ${mdb_operator_filename}"
kubectl -n "${operator_namespace}" get "deployment/${operator_name}" -o yaml > "${log_dir}/${mdb_operator_filename}"

echo "+ Kubernetes Version Reported by kubectl"
kubectl version

if type oc &> /dev/null; then
    echo "+ Kubernetes Version Reported by oc"
    oc version
fi

operator_logs_filename="${operator_name}_${current_date}.logs"
echo "+ Saving Operator logs to file ${operator_logs_filename}"
kubectl -n "${operator_namespace}" logs "deployment/${operator_name}" --tail 2000 > "${log_dir}/${operator_logs_filename}"

operator_container_pretty_name=$(kubectl -n "${operator_namespace}" exec -it "${mdb_operator_pod}" -- sh -c "cat /etc/*release" | grep "PRETTY_NAME" | cut -d'=' -f 2)
echo "+ Operator is running on: ${operator_container_pretty_name}"

if [ ${collect_om} == 0 ]
then
    database_container_pretty_name=$(kubectl -n "${namespace}" exec -it "${mdb_resource}-0" -- sh -c "cat /etc/*release" | grep "PRETTY_NAME" | cut -d'=' -f 2)
    echo "+ Database is running on: ${database_container_pretty_name}"
fi

echo "++ Kubernetes Cluster Ecosystem"
echo "+ Kubectl Cluster Information"
kubectl cluster-info

if [ ${private_mode} == 0 ]; then
    kubectl_cluster_info_filename="kubectl_cluster_info_${current_date}.logs"
    echo "+ Saving Cluster Info to file ${kubectl_cluster_info_filename} (this might take a few minutes)"
    kubectl cluster-info dump | gzip > "${log_dir}/${kubectl_cluster_info_filename}.gz"
else
    echo "= Skipping Kubectl cluster information dump, use --private to enable."
fi

kubectl_sc_dump_filename="kubectl_storage_class_${current_date}.yaml"
kubectl get storageclass -o yaml > "${log_dir}/${kubectl_sc_dump_filename}"

nodes_filename="nodes.yaml"
echo "+ Nodes"
kubectl get nodes

echo "+ Saving Nodes full state to ${nodes_filename}"
kubectl get nodes -o yaml > "${log_dir}/${nodes_filename}"

if [ ${collect_om} == 0 ]
then
    echo "++ MongoDB Resource Running Environment"
    crd_filename="crd_mdb.yaml"
    echo "+ Saving MDB Customer Resource Definition into ${crd_filename}"
    kubectl -n "${namespace}" get crd/mongodb.mongodb.com -o yaml > "${log_dir}/${crd_filename}"

    project_filename="project.yaml"
    mdb_resource_name="mdb/${mdb_resource}"
    project_name=$(kubectl -n "${namespace}" get "${mdb_resource_name}" -o jsonpath='{.spec.opsManager.configMapRef.name}')
    credentials_name=$(kubectl -n "${namespace}" get "${mdb_resource_name}" -o jsonpath='{.spec.credentials}')

    resource_filename="mdb_object_${mdb_resource}.yaml"
    echo "+ MongoDB Resource Status"
    kubectl -n  "${namespace}" get "${mdb_resource_name}" -o yaml > "${log_dir}/${resource_filename}"

    echo "+ Saving Project YAML file to ${project_filename}"
    kubectl -n "${namespace}" get "configmap/${project_name}" -o yaml > "${log_dir}/${project_filename}"

    credentials_user=$(kubectl -n "${namespace}" get "secret/${credentials_name}" -o jsonpath='{.data.user}' | base64 --decode)
    echo "+ User configured is (credentials.user): ${credentials_user}"

    echo "= To get the Secret Public API Key use: kubectl -n ${namespace} get secret/${credentials_name} -o jsonpath='{.data.publicApiKey}' | base64 --decode)"

    statefulset_filename="statefulset.yaml"
    echo "+ Saving StatefulSet state to ${statefulset_filename}"
    kubectl -n "${namespace}" get "sts/${mdb_resource}" -o yaml > "${log_dir}/${statefulset_filename}"

    echo "+ Deployment Pods"
    kubectl -n "${namespace}" get pods | grep -E "^${mdb_resource}-[0-9]+"

    echo "+ Saving Pods state to ${mdb_resource}-N.logs"
    pods_in_namespace=$(kubectl -n "${namespace}" get pods -o name | cut -d'/' -f 2 | grep -E "^${mdb_resource}-[0-9]+")
    for pod in ${pods_in_namespace}; do
        kubectl -n "${namespace}" logs "${pod}" -c ${mdb_container_name} --tail 2000 > "${log_dir}/${pod}.log"
        kubectl -n "${namespace}" get event --field-selector "involvedObject.name=${pod}" > "${log_dir}/${pod}_events.log"
    done

    echo "+ Persistent Volumes"
    kubectl -n "${namespace}" get pv

    echo "+ Persistent Volume Claims"
    kubectl -n "${namespace}" get pvc

    pv_filename="persistent_volumes.yaml"
    echo "+ Saving Persistent Volumes state to ${pv_filename}"
    kubectl -n "${namespace}" get pv -o yaml > "${log_dir}/${pv_filename}"

    pvc_filename="persistent_volume_claims.yaml"
    echo "+ Saving Persistent Volumes Claims state to ${pvc_filename}"
    kubectl -n "${namespace}" get pvc -o yaml > "${log_dir}/${pvc_filename}"

    services_filename="services.yaml"
    echo "+ Services"
    kubectl -n "${namespace}" get services

    echo "+ Saving Services state to ${services_filename}"
    kubectl -n "${namespace}" get services -o yaml > "${log_dir}/${services_filename}"

    echo "+ Saving Events for the Namespace"
    kubectl -n "${namespace}" get events > "${log_dir}/events.log"

    echo "+ Certificates (no private keys are captured)"
    csr_filename="csr.text"
    kubectl get csr | grep "${namespace}" || true
    echo "+ Saving Certificate state into ${csr_filename}"
    kubectl describe "$(kubectl get csr -o name | grep "${namespace}")" > "${log_dir}/${csr_filename}" || true

    echo "++ MongoDBUser Resource Status"
    mdbusers_filename="mdbu.yaml"
    kubectl -n "${namespace}" get mdbu
    echo "+ Saving MongoDBUsers to ${mdbusers_filename}"
    kubectl -n "${namespace}" get mdbu > "${log_dir}/${mdbusers_filename}"

    crdu_filename="crd_mdbu.yaml"
    echo "+ Saving MongoDBUser Customer Resource Definition into ${crdu_filename}"
    kubectl -n "${namespace}" get crd/mongodbusers.mongodb.com -o yaml > "${log_dir}/${crdu_filename}"
fi

if [ ${collect_om} == 1 ]
then
    ops_manager_filename="ops_manager.yaml"
    echo "+ Saving OpsManager Status"
    kubectl -n "${namespace}" get om -o wide
    echo "+ Saving OpsManager Status to ${ops_manager_filename}"
    kubectl -n "${namespace}" get om -o yaml > "${log_dir}/${ops_manager_filename}"
    echo "+ Saving Pods state to ${om_resource_name}-N.logs"
    pods_in_namespace=$(kubectl -n "${namespace}" get pods -o name -l "app=${om_resource_name}-svc" | cut -d'/' -f 2)
    for pod in ${pods_in_namespace}; do
        kubectl -n "${namespace}" logs "${pod}" --tail 2000 > "${log_dir}/${pod}.log"
        echo "Collecting Events: ${pod}"
        kubectl -n "${namespace}" get event --field-selector "involvedObject.name=${pod}" > "${log_dir}/${pod}_events.log"
    done
    echo "+ Saving AppDB Pods state to ${om_resource_name}-db-N-<container_name>.logs"
    pods_in_namespace=$(kubectl -n "${namespace}" get pods -o name -l "app=${om_resource_name}-db-svc" | cut -d'/' -f 2 )
    for pod in ${pods_in_namespace}; do
        kubectl -n "${namespace}" logs "${pod}" -c "mongod" --tail 2000 > "${log_dir}/${pod}-mongod.log"
        kubectl -n "${namespace}" logs "${pod}" -c "mongodb-agent" --tail 2000 > "${log_dir}/${pod}-mongodb-agent.log"
        kubectl -n "${namespace}" logs "${pod}" -c "mongodb-agent-monitoring" --tail 2000 > "${log_dir}/${pod}-mongodb-agent-monitoring.log"
        echo "Collecting Events: ${pod}"
        kubectl -n "${namespace}" get event --field-selector "involvedObject.name=${pod}" > "${log_dir}/${pod}_events.log"
    done
fi

echo "++ Compressing files"
compressed_logs_filename="${namespace}__${mdb_resource}__${current_date}.tar.gz"
tar -czf "${compressed_logs_filename}" -C "${log_dir}" .

echo "- All logs have been captured and compressed into the file ${compressed_logs_filename}."
echo "- If support is needed, please attach this file to an email to provide you with a better support experience."
echo "- If there are additional logs that your organization is capturing, they should be made available in case of a support request."
