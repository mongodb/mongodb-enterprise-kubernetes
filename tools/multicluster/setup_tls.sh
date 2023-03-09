#!/usr/bin/env bash

set -Eeou pipefail

# This script is intended for demoing and not for general customer usage. This script has no official MongoDB support and is not guaranteed to be maintained.
#
# This script requires having `mkcert` installed for creating a local CA
# Sample usage:
# ./setup_tls.sh mongodb multi-cluster-replica-set

namespace="${1:-mongodb}"
resource="${2:-multi-replica-set}"

# Install cert-manager
helm upgrade --install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --set installCRDs=true

# Setup local CA
mkcert -install

# Create CA secret in kubernetes
kubectl create secret tls ca-key-pair --cert="$(mkcert --CAROOT)/rootCA.pem" --key="$(mkcert --CAROOT)/rootCA-key.pem" -n "${namespace}"

# Download mongodb certs and append them to the local CA cert
openssl s_client -showcerts -verify 2 -connect downloads.mongodb.com:443 -servername downloads.mongodb.com </dev/null | awk '/BEGIN/,/END/{ if(/BEGIN/){a++}; out="cert"a".crt"; print >out}' || true
cat "$(mkcert --CAROOT)/rootCA.pem" cert1.crt cert2.crt cert3.crt cert4.crt >>ca-chain.crt

# Create CA certificates config map from certificate chain
kubectl create configmap issuer-ca --from-file=mms-ca.crt=ca-chain.crt --from-file=ca-pem=ca-chain.crt -n "${namespace}"

# Crete Issuer for certs
cat <<EOF | kubectl -n "${namespace}" apply -f -
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: mongodb-ca-issuer
spec:
  ca:
    secretName: ca-key-pair
EOF

# Create server certificates on central cluster
cat <<EOF | kubectl -n "${namespace}" apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: clustercert-${resource}-cert
spec:
  dnsNames:
  - ${resource}-svc.mongodb.svc.cluster.local
  - ${resource}-0-0-svc.mongodb.svc.cluster.local
  - ${resource}-0-1-svc.mongodb.svc.cluster.local
  - ${resource}-0-2-svc.mongodb.svc.cluster.local
  - ${resource}-1-0-svc.mongodb.svc.cluster.local
  - ${resource}-1-1-svc.mongodb.svc.cluster.local
  - ${resource}-2-0-svc.mongodb.svc.cluster.local
  - ${resource}-2-1-svc.mongodb.svc.cluster.local
  - ${resource}-2-2-svc.mongodb.svc.cluster.local
  duration: 240h0m0s
  issuerRef:
    kind: Issuer
    name: mongodb-ca-issuer
  renewBefore: 120h0m0s
  secretName: clustercert-${resource}-cert
  subject:
    countries:
    - US
    localities:
    - NY
    organizationalUnits:
    - mongodb
    organizations:
    - cluster.local-server
    provinces:
    - NY
  usages:
  - digital signature
  - key encipherment
  - client auth
  - server auth
EOF

# Enable TLS for custom resource
kubectl -n "${namespace}" patch mdbm "${resource}" --type=json -p='[{"op": "add", "path": "/spec/security", "value": {"certsSecretPrefix": "clustercert", "tls": {"ca": "issuer-ca"}}}]'
