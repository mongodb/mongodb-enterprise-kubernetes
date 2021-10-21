#!/bin/bash

set -eux

# change the clusternames as per your need
export CTX_CLUSTER1=gke_k8s-rdas_us-east1-b_member-1a
export CTX_CLUSTER2=gke_k8s-rdas_us-east1-c_member-2a
export CTX_CLUSTER3=gke_k8s-rdas_us-west1-a_member-3a
export VERSION=1.10.3

# download Istio 1.10.3 under the path
curl -L https://istio.io/downloadIstio | ISTIO_VERSION=${VERSION} sh -

# checks if external IP has been assigned to a service object, in our case we are interested in east-west gateway
function_check_external_ip_assigned() {
 while : ; do
   ip=$(kubectl --context="$1" get svc istio-eastwestgateway -n istio-system --output jsonpath='{.status.loadBalancer.ingress[0].ip}')  
   if [ -n "$ip" ]
   then 
     echo "external ip assigned $ip"
     break
   else 
     echo "waiting for external ip to be assigned"
   fi
done
}

cd istio-${VERSION}
mkdir -p certs
pushd certs

# create root trust for the clusters
make -f ../tools/certs/Makefile.selfsigned.mk root-ca
make -f ../tools/certs/Makefile.selfsigned.mk ${CTX_CLUSTER1}-cacerts
make -f ../tools/certs/Makefile.selfsigned.mk ${CTX_CLUSTER2}-cacerts
make -f ../tools/certs/Makefile.selfsigned.mk ${CTX_CLUSTER3}-cacerts

kubectl --context="${CTX_CLUSTER1}" create ns istio-system
kubectl --context="${CTX_CLUSTER1}" create secret generic cacerts -n istio-system \
      --from-file=${CTX_CLUSTER1}/ca-cert.pem \
      --from-file=${CTX_CLUSTER1}/ca-key.pem \
      --from-file=${CTX_CLUSTER1}/root-cert.pem \
      --from-file=${CTX_CLUSTER1}/cert-chain.pem

kubectl --context="${CTX_CLUSTER2}" create ns istio-system
kubectl --context="${CTX_CLUSTER2}" create secret generic cacerts -n istio-system \
      --from-file=${CTX_CLUSTER2}/ca-cert.pem \
      --from-file=${CTX_CLUSTER2}/ca-key.pem \
      --from-file=${CTX_CLUSTER2}/root-cert.pem \
      --from-file=${CTX_CLUSTER2}/cert-chain.pem

kubectl --context="${CTX_CLUSTER3}" create ns istio-system
kubectl --context="${CTX_CLUSTER3}" create secret generic cacerts -n istio-system \
      --from-file=${CTX_CLUSTER3}/ca-cert.pem \
      --from-file=${CTX_CLUSTER3}/ca-key.pem \
      --from-file=${CTX_CLUSTER3}/root-cert.pem \
      --from-file=${CTX_CLUSTER3}/cert-chain.pem
popd

# label namespace in cluster1
kubectl --context="${CTX_CLUSTER1}" get namespace istio-system && \
  kubectl --context="${CTX_CLUSTER1}" label namespace istio-system topology.istio.io/network=network1

cat <<EOF > cluster1.yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  values:
    global:
      meshID: mesh1
      multiCluster:
        clusterName: cluster1
      network: network1
EOF
bin/istioctl install --context="${CTX_CLUSTER1}" -f cluster1.yaml
samples/multicluster/gen-eastwest-gateway.sh \
    --mesh mesh1 --cluster cluster1 --network network1 | \
    bin/istioctl --context="${CTX_CLUSTER1}" install -y -f -


# check if external IP is assigned to east-west gateway in cluster1
function_check_external_ip_assigned "${CTX_CLUSTER1}"


# expose services in cluster1
kubectl --context="${CTX_CLUSTER1}" apply -n istio-system -f \
    samples/multicluster/expose-services.yaml


kubectl --context="${CTX_CLUSTER2}" get namespace istio-system && \
  kubectl --context="${CTX_CLUSTER2}" label namespace istio-system topology.istio.io/network=network2


cat <<EOF > cluster2.yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  values:
    global:
      meshID: mesh1
      multiCluster:
        clusterName: cluster2
      network: network2
EOF

bin/istioctl install --context="${CTX_CLUSTER2}" -f cluster2.yaml

samples/multicluster/gen-eastwest-gateway.sh \
    --mesh mesh1 --cluster cluster2 --network network2 | \
    bin/istioctl --context="${CTX_CLUSTER2}" install -y -f -

# check if external IP is assigned to east-west gateway in cluster2
function_check_external_ip_assigned "${CTX_CLUSTER2}"

kubectl --context="${CTX_CLUSTER2}" apply -n istio-system -f \
    samples/multicluster/expose-services.yaml

# cluster3
kubectl --context="${CTX_CLUSTER3}" get namespace istio-system && \
  kubectl --context="${CTX_CLUSTER3}" label namespace istio-system topology.istio.io/network=network3

cat <<EOF > cluster3.yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  values:
    global:
      meshID: mesh1
      multiCluster:
        clusterName: cluster3
      network: network3
EOF

bin/istioctl install --context="${CTX_CLUSTER3}" -f cluster3.yaml

samples/multicluster/gen-eastwest-gateway.sh \
    --mesh mesh1 --cluster cluster3 --network network3 | \
    bin/istioctl --context="${CTX_CLUSTER3}" install -y -f -


# check if external IP is assigned to east-west gateway in cluster3
function_check_external_ip_assigned "${CTX_CLUSTER3}"

kubectl --context="${CTX_CLUSTER3}" apply -n istio-system -f \
    samples/multicluster/expose-services.yaml


# enable endpoint discovery
bin/istioctl x create-remote-secret \
  --context="${CTX_CLUSTER1}" \
  -n istio-system \
  --name=cluster1 | \
  kubectl apply -f - --context="${CTX_CLUSTER2}"

bin/istioctl x create-remote-secret \
  --context="${CTX_CLUSTER1}" \
  -n istio-system \
  --name=cluster1 | \
  kubectl apply -f - --context="${CTX_CLUSTER3}"

bin/istioctl x create-remote-secret \
  --context="${CTX_CLUSTER2}" \
  -n istio-system \
  --name=cluster2 | \
  kubectl apply -f - --context="${CTX_CLUSTER1}"

bin/istioctl x create-remote-secret \
  --context="${CTX_CLUSTER2}" \
  -n istio-system \
  --name=cluster2 | \
  kubectl apply -f - --context="${CTX_CLUSTER3}"

bin/istioctl x create-remote-secret \
  --context="${CTX_CLUSTER3}" \
  -n istio-system \
  --name=cluster3 | \
  kubectl apply -f - --context="${CTX_CLUSTER1}"

bin/istioctl x create-remote-secret \
  --context="${CTX_CLUSTER3}" \
  -n istio-system \
  --name=cluster3 | \
  kubectl apply -f - --context="${CTX_CLUSTER2}"

  # cleanup: delete the istio repo at the end
cd ..
rm -r istio-${VERSION}
rm -f cluster1.yaml cluster2.yaml cluster3.yaml 
