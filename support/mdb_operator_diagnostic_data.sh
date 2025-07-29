#!/usr/bin/env bash

set -Eeou pipefail

# Check if oc and kubectl exist
command -v oc >/dev/null 2>&1
ocExists=$?

command -v kubectl >/dev/null 2>&1
kcExists=$?

# Let the user override via environment variable
dToolUser=${DTOOL:-""}
dToolUserArgs=${DTOOLARGS:-""}


if [[ -n "$dToolUser" ]]; then
    echo "[INFO] DTOOL set by user: $dToolUser"
    if command -v "$dToolUser" >/dev/null 2>&1; then
        dTool="$dToolUser"
    else
        echo "Error: $dToolUser is not installed or not in PATH." >&2
        exit 1
    fi
else
    echo "[INFO] No DTOOL set by user."

    if [[ $ocExists -eq 0 && $kcExists -eq 0 ]]; then
        echo "[INFO] Both oc and kubectl are available. Defaulting to oc."
        dTool="oc"
    elif [[ $ocExists -eq 0 ]]; then
        echo "[INFO] Only oc found. Using oc."
        dTool="oc"
    elif [[ $kcExists -eq 0 ]]; then
        echo "[INFO] Only kubectl found. Using kubectl."
        dTool="kubectl"
    else
        echo "Error: Neither oc nor kubectl is available. Install at least one of them." >&2
        exit 1
    fi
fi

echo "[INFO] Using $dTool to run commands."
#
# mdb_operator_diagnostic_data.sh
#
# Use this script to gather data about your MongoDB Enterprise Kubernetes Operator
# and the MongoDB Resources deployed with it.
#

usage() {
  local script_name
  script_name=$(basename "${0}")
  
  echo "------------------------------------------------------------------------------"
  echo "Usage:"
  echo "${script_name} <namespace> <mdb/om_resource_name> [<operator_namespace>] [<operator_name>] [--om] [--private]"
  echo "------------------------------------------------------------------------------"
  echo "#Scenario 01: Collecting MongoDB Logs (Operator in same namespace as MongoDB):"
  echo "------------------------------------------------------------------------------"
  echo "Example: Operator_Namespace: mongodb, Deployment_Namespace: mongodb, Deployment_Name: myreplicaset"
  echo "Usage: ${script_name} mongodb myreplicaset"
  echo "For OpenShift: ${script_name} mongodb myreplicaset mongodb enterprise-operator"
  echo "------------------------------------------------------------------------------"
  echo "#Scenario 02: Collecting MongoDB Logs (Operator in different namespace as MongoDB):"
  echo "------------------------------------------------------------------------------"
  echo "Example: Operator_Namespace: mdboperator, Deployment_Namespace: mongodb, Deployment_Name: myreplicaset"
  echo "Usage: ${script_name} mongodb myreplicaset mdboperator"
  echo "For OpenShift: ${script_name} mongodb myreplicaset mdboperator enterprise-operator"
  echo "------------------------------------------------------------------------------"
  echo "#Scenario 03: Collecting Ops Manager Logs (Operator in same namespace as Ops Manager):"
  echo "------------------------------------------------------------------------------"
  echo "Example: Operator_Namespace: mongodb, Deployment_Namespace: mongodb, Deployment_Name: ops-manager"
  echo "Usage: ${script_name} mongodb ops-manager --om"
  echo "For OpenShift: ${script_name} mongodb ops-manager mongodb enterprise-operator --om"
  echo "------------------------------------------------------------------------------"
  echo "#Scenario 04: Collecting Ops Manager Logs (Operator in different namespace as Ops Manager):"
  echo "Example: Operator_Namespace: mdboperator, Deployment_Namespace: mongodb, Deployment_Name: ops-manager"
  echo "Usage: ${script_name} mongodb ops-manager mdboperator --om"
  echo "For OpenShift: ${script_name} mongodb ops-manager mdboperator enterprise-operator --om"
  echo "------------------------------------------------------------------------------"
  echo "#Scenario 05: Collecting MongoDB Logs but for multi-cluster (Operator is in a different namespace as MongoDB). Note: Member Cluster and Central Cluster is now given via an environment Flag:"
  echo 'Example: MEMBER_CLUSTERS:"kind-e2e-cluster-1 kind-e2e-cluster-2 kind-e2e-cluster-3", CENTRAL_CLUSTER:"kind-e2e-operator", Operator_Namespace: mdboperator, Deployment_Namespace: mongodb, Deployment_Name: multi-replica-set'
  echo "Usage:  MEMBER_CLUSTERS='kind-e2e-cluster-1 kind-e2e-cluster-2 kind-e2e-cluster-3' CENTRAL_CLUSTER='kind-e2e-operator' ${script_name} mongodb multi-replica-set mdboperator enterprise-operator"
  echo "For OpenShift:  MEMBER_CLUSTERS='kind-e2e-cluster-1 kind-e2e-cluster-2 kind-e2e-cluster-3' CENTRAL_CLUSTER='kind-e2e-operator' ${script_name} mongodb multi-replica-set mdboperator enterprise-operator"
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

collect_om=0
contains "--om" "$@" && collect_om=1

if [ ${collect_om} == 1 ]; then
  if [[ $3 == "--om" ]]; then
    operator_namespace="${1}"
    operator_name="mongodb-enterprise-operator"
    om_resource_name="${2}"
  elif [[ $4 == "--om" ]]; then
    operator_namespace="${3}"
    operator_name="mongodb-enterprise-operator"
    om_resource_name="${2}"
  elif [[ $5 == "--om" ]]; then
    operator_namespace="${3}"
    operator_name="$4"
    om_resource_name="${2}"
  fi
else
  operator_name="${4:-mongodb-enterprise-operator}"
  operator_namespace="${3:-$1}"
fi

dump_all() {
  local central=${1}
  if [ "${central}" == "member" ]; then
    if [ "${collect_om}" == 0 ]; then
      pod_name=$($dTool $dToolUserArgs get pods -l controller -n "${namespace}" --no-headers=true | awk '{print $1}' | head -n 1)
      database_container_pretty_name=$($dTool $dToolUserArgs -n "${namespace}" exec -it "${pod_name}" -- sh -c "cat /etc/*release" | grep "PRETTY_NAME" | cut -d'=' -f 2)
      echo "+ Database is running on: ${database_container_pretty_name}"
    fi

    statefulset_filename="statefulset.yaml"
    echo "+ Saving StatefulSet state to ${statefulset_filename}"
    $dTool $dToolUserArgs -n "${namespace}" -l controller get "sts" -o yaml >"${log_dir}/${statefulset_filename}"

    echo "+ Deployment Pods"
    $dTool $dToolUserArgs -n "${namespace}" get pods | grep -E "^${mdb_resource}-+"

    echo "+ Saving Pods state to ${mdb_resource}-N.logs"
    pods_in_namespace=$($dTool $dToolUserArgs get pods --namespace "${namespace}" --selector=controller=mongodb-enterprise-operator --no-headers -o custom-columns=":metadata.name")

    mdb_container_name="mongodb-enterprise-database"
    for pod in ${pods_in_namespace}; do
      $dTool $dToolUserArgs -n "${namespace}" logs "${pod}" -c ${mdb_container_name} --tail 2000 >"${log_dir}/${pod}.log"
      $dTool $dToolUserArgs -n "${namespace}" get event --field-selector "involvedObject.name=${pod}" >"${log_dir}/${pod}_events.log"
    done

    echo "+ Persistent Volumes"
    $dTool $dToolUserArgs -n "${namespace}" get pv

    echo "+ Persistent Volume Claims"
    $dTool $dToolUserArgs -n "${namespace}" get pvc

    pv_filename="persistent_volumes.yaml"
    echo "+ Saving Persistent Volumes state to ${pv_filename}"
    $dTool $dToolUserArgs -n "${namespace}" get pv -o yaml >"${log_dir}/${pv_filename}"

    pvc_filename="persistent_volume_claims.yaml"
    echo "+ Saving Persistent Volumes Claims state to ${pvc_filename}"
    $dTool $dToolUserArgs -n "${namespace}" get pvc -o yaml >"${log_dir}/${pvc_filename}"

    services_filename="services.yaml"
    echo "+ Services"
    $dTool $dToolUserArgs -n "${namespace}" get services

    echo "+ Saving Services state to ${services_filename}"
    $dTool $dToolUserArgs -n "${namespace}" get services -o yaml >"${log_dir}/${services_filename}"

    echo "+ Saving Events for the Namespace"
    $dTool $dToolUserArgs -n "${namespace}" get events >"${log_dir}/events.log"

  else
    echo "++ MongoDB Resource Running Environment"

    if [ -z "${CENTRAL_CLUSTER}" ]; then
      crd_filename="crd_mdb.yaml"
      echo "+ Saving MDB Customer Resource Definition into ${crd_filename}"
      $dTool $dToolUserArgs -n "${operator_namespace}" get crd/mongodb.mongodb.com -o yaml >"${log_dir}/${crd_filename}"
      mdb_resource_name="mdb/${mdb_resource}"
      resource_filename="mdb_object_${mdb_resource}.yaml"
    else
      crd_filename="crd_mdbmc.yaml"
      echo "+ Saving MDBMC Customer Resource Definition into ${crd_filename}"
      echo $dTool $dToolUserArgs -n "${operator_namespace}" get crd/mongodbmulticluster.mongodb.com -o yaml >"${log_dir}/${crd_filename}"
      mdb_resource_name="mdbmc/${mdb_resource}"
      resource_filename="mdbmc_object_${mdb_resource}.yaml"
    fi

    project_filename="project.yaml"
    project_name=$($dTool $dToolUserArgs -n "${namespace}" get "${mdb_resource_name}" -o jsonpath='{.spec.opsManager.configMapRef.name}')
    credentials_name=$($dTool $dToolUserArgs -n "${namespace}" get "${mdb_resource_name}" -o jsonpath='{.spec.credentials}')

    echo "+ MongoDB Resource Status"
    $dTool $dToolUserArgs -n "${namespace}" get "${mdb_resource_name}" -o yaml >"${log_dir}/${resource_filename}"

    echo "+ Saving Project YAML file to ${project_filename}"
    $dTool $dToolUserArgs -n "${namespace}" get "configmap/${project_name}" -o yaml >"${log_dir}/${project_filename}"
    credentials_user=$($dTool $dToolUserArgs -n "${namespace}" get "secret/${credentials_name}" -o jsonpath='{.data.user}' | base64 --decode)
    echo "+ User configured is (credentials.user): ${credentials_user}"

    echo "= To get the Secret Public API Key use: $dTool $dToolUserArgs -n ${namespace} get secret/${credentials_name} -o jsonpath='{.data.publicApiKey}' | base64 --decode)"

    echo "+ Certificates (no private keys are captured)"
    csr_filename="csr.text"
    $dTool $dToolUserArgs get csr | grep "${namespace}" || true
    echo "+ Saving Certificate state into ${csr_filename}"
    $dTool $dToolUserArgs describe "$($dTool $dToolUserArgs get csr -o name | grep "${namespace}")" >"${log_dir}/${csr_filename}" || true

    echo "++ MongoDBUser Resource Status"
    mdbusers_filename="mdbu.yaml"
    $dTool $dToolUserArgs -n "${namespace}" get mdbu
    echo "+ Saving MongoDBUsers to ${mdbusers_filename}"
    $dTool $dToolUserArgs -n "${namespace}" get mdbu >"${log_dir}/${mdbusers_filename}"

    crdu_filename="crd_mdbu.yaml"
    echo "+ Saving MongoDBUser Customer Resource Definition into ${crdu_filename}"
    $dTool $dToolUserArgs -n "${namespace}" get crd/mongodbusers.mongodb.com -o yaml >"${log_dir}/${crdu_filename}"
  fi

}

MEMBER_CLUSTERS=${MEMBER_CLUSTERS:-}
CENTRAL_CLUSTER=${CENTRAL_CLUSTER:-}

current_date="$(date +%Y-%m-%d_%H_%M)"

private_mode=1
contains "--private" "$@" && private_mode=0

log_dir="logs_${current_date}"
mkdir -p "${log_dir}" &>/dev/null

if [ -n "${CENTRAL_CLUSTER}" ]; then
  if [ -z "${MEMBER_CLUSTERS}" ]; then
    echo "CENTRAL_CLUSTER is set but no MEMBER_CLUSTERS"
    exit 1
  else
    echo "starting with the CENTRAL_CLUSTER!"
    $dTool $dToolUserArgs config use-context "${CENTRAL_CLUSTER}"
  fi
fi

if [ -n "${MEMBER_CLUSTERS}" ]; then
  if [ -z "${CENTRAL_CLUSTER}" ]; then
    echo "MEMBER_CLUSTERS is set but no CENTRAL_CLUSTER"
    exit 1
  fi
fi

if ! $dTool $dToolUserArgs get "namespace/${namespace}" &>/dev/null; then
  echo "Error fetching namespace. Make sure name ${namespace} for Namespace is correct."
  exit 1
fi

if [ ${collect_om} == 0 ]; then
  if [ -z "${CENTRAL_CLUSTER}" ]; then
    if ! $dTool $dToolUserArgs -n "${namespace}" get "mdb/${mdb_resource}" &>/dev/null; then
      echo "Error fetching the MongoDB resource. Make sure the '${namespace}/${mdb_resource}'  is correct."
      exit 1
    fi
  else
    if ! $dTool $dToolUserArgs -n "${namespace}" get "mdbmc/${mdb_resource}" &>/dev/null; then
      echo "Error fetching the MongoDB MultiCluster resource. Make sure the '${namespace}/${mdb_resource}' is correct."
      exit 1
    fi
  fi
fi

echo "++ Versions"
mdb_operator_pod=$($dTool $dToolUserArgs -n "${operator_namespace}" get pods -l "app.kubernetes.io/component=controller" -o name | cut -d'/' -f 2)
echo "${operator_namespace}"
echo "+ Operator Pod: pod/${mdb_operator_pod}"

if ! $dTool $dToolUserArgs -n "${operator_namespace}" get "pod/${mdb_operator_pod}" &>/dev/null; then
  echo "Error fetching the MongoDB Operator Deployment. Make sure the pod/${mdb_operator_pod} exist and it is running."
  exit 1
fi

if ! $dTool $dToolUserArgs -n "${namespace}" get om -o wide &>/dev/null; then
  echo "Error fetching the MongoDB OpsManager Resource."
fi

if [ ${private_mode} == 0 ]; then
  echo "+ Running on private mode. Make sure you don't share the results of this run outside your organization."
fi

mdb_operator_filename="operator.yaml"
echo "+ Saving Operator Deployment into ${mdb_operator_filename}"
$dTool $dToolUserArgs -n "${operator_namespace}" get "pod/${mdb_operator_pod}" -o yaml >"${log_dir}/${mdb_operator_filename}"

echo "+ Kubernetes Version Reported by $dTool"
$dTool $dToolUserArgs version

if type oc &>/dev/null; then
  echo "+ Kubernetes Version Reported by oc"
  oc version
fi

operator_logs_filename="${operator_name}_${current_date}.logs"
echo "+ Saving Operator logs to file ${operator_logs_filename}"
$dTool $dToolUserArgs -n "${operator_namespace}" logs "pod/${mdb_operator_pod}" --tail 2000 >"${log_dir}/${operator_logs_filename}"

operator_container_pretty_name=$($dTool $dToolUserArgs -n "${operator_namespace}" exec -it "${mdb_operator_pod}" -- sh -c "cat /etc/*release" | grep "PRETTY_NAME" | cut -d'=' -f 2)
echo "+ Operator is running on: ${operator_container_pretty_name}"

echo "++ Kubernetes Cluster Ecosystem"
echo "+ $dTool Cluster Information"
$dTool $dToolUserArgs cluster-info

if [ ${private_mode} == 0 ]; then
  $dTool_cluster_info_filename="$dTool_cluster_info_${current_date}.logs"
  echo "+ Saving Cluster Info to file ${$dTool_cluster_info_filename} (this might take a few minutes)"
  $dTool $dToolUserArgs cluster-info dump | gzip >"${log_dir}/${$dTool_cluster_info_filename}.gz"
else
  echo "= Skipping $dTool cluster information dump, use --private to enable."
fi

dTool_sc_dump_filename="${dTool}_storage_class_${current_date}.yaml"
$dTool $dToolUserArgs get storageclass -o yaml >"${log_dir}/${dTool_sc_dump_filename}"

nodes_filename="nodes.yaml"
echo "+ Nodes"
$dTool $dToolUserArgs get nodes

echo "+ Saving Nodes full state to ${nodes_filename}"
$dTool $dToolUserArgs get nodes -o yaml >"${log_dir}/${nodes_filename}"

if [ ${collect_om} == 0 ]; then
  if [ -n "${CENTRAL_CLUSTER}" ]; then
    for member_cluster in ${MEMBER_CLUSTERS}; do
      echo "Dumping diagnostics for context ${member_cluster}"
      $dTool $dToolUserArgs config use-context "${member_cluster}"
      dump_all "member"
    done
    echo "Dumping diagnostics for context ${CENTRAL_CLUSTER}"
    $dTool $dToolUserArgs config use-context "${CENTRAL_CLUSTER}"
    dump_all "central"
  else
    dump_all "member"
    dump_all "central"
  fi
fi

if [ ${collect_om} == 1 ]; then
  ops_manager_filename="ops_manager.yaml"
  echo "+ Saving OpsManager Status"
  $dTool $dToolUserArgs -n "${namespace}" get om -o wide
  echo "+ Saving OpsManager Status to ${ops_manager_filename}"
  $dTool $dToolUserArgs -n "${namespace}" get om -o yaml >"${log_dir}/${ops_manager_filename}"
  echo "+ Saving Pods state to ${om_resource_name}-N.logs"
  pods_in_namespace=$($dTool $dToolUserArgs -n "${namespace}" get pods -o name -l "app=${om_resource_name}-svc" | cut -d'/' -f 2)
  for pod in ${pods_in_namespace}; do
    $dTool $dToolUserArgs -n "${namespace}" logs "${pod}" --tail 2000 >"${log_dir}/${pod}.log"
    echo "Collecting Events: ${pod}"
    $dTool $dToolUserArgs -n "${namespace}" get event --field-selector "involvedObject.name=${pod}" >"${log_dir}/${pod}_events.log"
  done
  echo "+ Saving AppDB Pods state to ${om_resource_name}-db-N-<container_name>.logs"
  pods_in_namespace=$($dTool $dToolUserArgs -n "${namespace}" get pods -o name -l "app=${om_resource_name}-db-svc" | cut -d'/' -f 2)
  for pod in ${pods_in_namespace}; do
    $dTool $dToolUserArgs -n "${namespace}" logs "${pod}" -c "mongod" --tail 2000 >"${log_dir}/${pod}-mongod.log"
    $dTool $dToolUserArgs -n "${namespace}" logs "${pod}" -c "mongodb-agent" --tail 2000 >"${log_dir}/${pod}-mongodb-agent.log"
    $dTool $dToolUserArgs -n "${namespace}" logs "${pod}" -c "mongodb-agent-monitoring" --tail 2000 >"${log_dir}/${pod}-mongodb-agent-monitoring.log"
    echo "Collecting Events: ${pod}"
    $dTool $dToolUserArgs -n "${namespace}" get event --field-selector "involvedObject.name=${pod}" >"${log_dir}/${pod}_events.log"
  done
fi

echo "++ Compressing files"
compressed_logs_filename="${namespace}__${mdb_resource}__${current_date}.tar.gz"
tar -czf "${compressed_logs_filename}" -C "${log_dir}" .

echo "- All logs have been captured and compressed into the file ${compressed_logs_filename}."
echo "- If support is needed, please attach this file to an email to provide you with a better support experience."
echo "- If there are additional logs that your organization is capturing, they should be made available in case of a support request."
