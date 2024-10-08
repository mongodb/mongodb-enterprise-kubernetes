#!/usr/bin/env bash

set -eou pipefail

source ../../scripts/sample_test_runner.sh

prepare_snippets

run 0010_create_gke_cluster_0.sh &
run 0010_create_gke_cluster_1.sh &
run 0010_create_gke_cluster_2.sh &
wait
run 0011_gcloud_set_current_project.sh
run 0020_get_gke_credentials.sh
run_for_output 0030_verify_access_to_clusters.sh

run 0040_install_istio.sh

run 0045_create_operator_namespace.sh
run 0045_create_ops_manager_namespace.sh

run 0046_create_image_pull_secrets.sh

run 0050_check_cluster_connectivity_create_sts_0.sh
run 0050_check_cluster_connectivity_create_sts_1.sh
run 0050_check_cluster_connectivity_create_sts_2.sh
run 0060_check_cluster_connectivity_wait_for_sts.sh
run 0070_check_cluster_connectivity_create_pod_service_0.sh
run 0070_check_cluster_connectivity_create_pod_service_1.sh
run 0070_check_cluster_connectivity_create_pod_service_2.sh
run 0080_check_cluster_connectivity_create_round_robin_service_0.sh
run 0080_check_cluster_connectivity_create_round_robin_service_1.sh
run 0080_check_cluster_connectivity_create_round_robin_service_2.sh
run_for_output 0090_check_cluster_connectivity_verify_pod_0_0_from_cluster_1.sh
run_for_output 0090_check_cluster_connectivity_verify_pod_1_0_from_cluster_0.sh
run_for_output 0090_check_cluster_connectivity_verify_pod_1_0_from_cluster_2.sh
run_for_output 0090_check_cluster_connectivity_verify_pod_2_0_from_cluster_0.sh
run 0100_check_cluster_connectivity_cleanup.sh

run_for_output 0200_kubectl_mongodb_configure_multi_cluster.sh
run_for_output 0205_helm_configure_repo.sh
run_for_output 0210_helm_install_operator.sh
run_for_output 0211_check_operator_deployment.sh

run 0250_generate_certs.sh
run 0255_create_cert_secrets.sh

run 0300_ops_manager_create_admin_credentials.sh
run 0310_ops_manager_deploy_on_single_member_cluster.sh
run_for_output 0311_ops_manager_wait_for_pending_state.sh
run_for_output 0312_ops_manager_wait_for_running_state.sh
run 0320_ops_manager_add_second_cluster.sh
run_for_output 0321_ops_manager_wait_for_pending_state.sh
run_for_output 0322_ops_manager_wait_for_running_state.sh

run 0400_install_minio_s3.sh
run 0500_ops_manager_prepare_s3_backup_secrets.sh
run 0510_ops_manager_enable_s3_backup.sh
run_for_output 0522_ops_manager_wait_for_running_state.sh
