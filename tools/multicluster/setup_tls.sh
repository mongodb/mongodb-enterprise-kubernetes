#!/usr/bin/env bash

set -Eeou pipefail

# this is useful for the user to see what is actually executing here
set -x

# This script is intended for demoing and not for general customer usage. This script has no official MongoDB support and is not guaranteed to be maintained.
#
# This script:
#   - requires having "mkcert" (https://github.com/FiloSottile/mkcert) installed for creating a local CA.
#   - executes all operation in the current kubectl context
#   - installs cert-manager in cert-manager namespace using helm
#   - creates issuer CA secret "ca-key-pair" using mkcert's root CA key pair to create ClusterIssuer in cert-manager
#   - creates "issuer-ca" config map with the all necessary CA certificates for MongoDB resources
#   - creates ClusterIssuer in cert-manager to issue certificates in different namespaces
#   - creates Certificate in cert-manager to issue the certificate in the desired namespace. Cert-manager will create a secret in the specified namespace named: "certprefix-${resource}-cert".
#   - tries to configure TLS encryption in MongoDBMultiCluster resource
#   - It is issued for a wildcard hostname "*.${namespace}.svc.cluster.local" in SAN field, and it's suitable to use in all MongoDB databases and as Ops Manager's server certificate.
# Sample usage:
# ./setup_tls.sh mongodb multi-cluster-replica-set

namespace="${1:-mongodb}"
resource="${2:-multi-replica-set}"

# Install cert-manager
helm upgrade --install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --set installCRDs=true

# Setup local CA
echo "Installing root CA using: mkcert -install. Sudo password might be required."
mkcert -install

# Create CA secret in kubernetes
kubectl create secret tls ca-key-pair --cert="$(mkcert --CAROOT)/rootCA.pem" --key="$(mkcert --CAROOT)/rootCA-key.pem" -n cert-manager || true

# Download mongodb certs and append them to the local CA cert
openssl s_client -showcerts -verify 2 -connect downloads.mongodb.com:443 -servername downloads.mongodb.com </dev/null | awk '/BEGIN/,/END/{ if(/BEGIN/){a++}; out="cert"a".crt"; print >out}'
cat "$(mkcert --CAROOT)/rootCA.pem" cert1.crt cert2.crt cert3.crt cert4.crt >ca-chain.crt

# Create CA certificates config map from certificate chain
kubectl create configmap issuer-ca --from-file=mms-ca.crt=ca-chain.crt --from-file=ca-pem=ca-chain.crt -n "${namespace}"

# Create ClusterIssuer for certs
cat <<EOF | kubectl -n "${namespace}" apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
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
  - "*.${namespace}.svc.cluster.local"
  duration: 240h0m0s
  issuerRef:
    kind: ClusterIssuer
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
kubectl -n "${namespace}" patch mdbm "${resource}" --type=json -p='[{"op": "add", "path": "/spec/security", "value": {"certsSecretPrefix": "clustercert", "tls": {"ca": "issuer-ca"}}}]' || {
  echo "Couldn't enable TLS in MongoDBMultiCluster resource ${resource}"
}
