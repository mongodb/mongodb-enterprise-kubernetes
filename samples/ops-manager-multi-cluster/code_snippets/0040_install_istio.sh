CTX_CLUSTER1=${K8S_CLUSTER_0_CONTEXT_NAME} \
CTX_CLUSTER2=${K8S_CLUSTER_1_CONTEXT_NAME} \
CTX_CLUSTER3=${K8S_CLUSTER_2_CONTEXT_NAME} \
ISTIO_VERSION="1.20.2" \
../multi-cluster/install_istio_separate_network.sh
