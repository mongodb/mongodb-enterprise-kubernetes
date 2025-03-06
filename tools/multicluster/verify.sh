#!/usr/bin/env bash

set -euo pipefail

# Verify the signature of a binary with the operator's public key
# goreleaser takes care of calling this script as a hook.

ARTIFACT=$1
SIGNATURE="${ARTIFACT}.sig"

HOSTED_SIGN_PUBKEY="https://cosign.mongodb.com/mongodb-enterprise-kubernetes-operator.pem" # to complete
TMPDIR=${TMPDIR:-/tmp}
KEY_FILE="${TMPDIR}/host-public.key"
SIGNING_IMAGE_URI=${SIGNING_IMAGE_URI}

curl -o ${KEY_FILE} "${HOSTED_SIGN_PUBKEY}"
echo "Verifying signature ${SIGNATURE} of artifact ${ARTIFACT}"
echo "Keyfile is ${KEY_FILE}"

# When working locally, the following command can be used instead of Docker
# cosign verify-blob --key ${KEY_FILE} --signature ${SIGNATURE} ${ARTIFACT}

docker run \
  --rm \
  -v $(pwd):$(pwd) \
  -v ${KEY_FILE}:${KEY_FILE} \
  -w $(pwd) \
  ${SIGNING_IMAGE_URI} \
  cosign verify-blob --key ${KEY_FILE} --signature ${SIGNATURE} ${ARTIFACT}

# Without below line, Evergreen fails at archiving with "open dist/kubectl-[...]/kubectl-mongodb.sig: permission denied
sudo chmod 666 ${SIGNATURE}
