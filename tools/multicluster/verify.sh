#!/bin/bash

set -euo pipefail

ARTIFACT=$1
SIGNATURE="${ARTIFACT}.sig"

HOSTED_SIGN_PUBKEY="https://cosign.mongodb.com/mongodb-enterprise-kubernetes-operator.pem" # to complete
TMPDIR=${TMPDIR:-/tmp}
KEY_FILE="${TMPDIR}/host-public.key"

curl -o ${KEY_FILE} "${HOSTED_SIGN_PUBKEY}"
echo "Verifying signature ${SIGNATURE} of artifact ${ARTIFACT}"
echo "Keyfile is ${KEY_FILE}"
cosign verify-blob --key ${KEY_FILE} --signature ${SIGNATURE} ${ARTIFACT}
