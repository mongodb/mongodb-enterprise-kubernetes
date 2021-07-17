#
# Base Template Dockerfile for Operator Image.
#

ARG imagebase
FROM ${imagebase} as base

FROM ubuntu:xenial-20210416


LABEL name="MongoDB Enterprise Operator" \
      maintainer="support@mongodb.com" \
      vendor="MongoDB" \
      version="1.10.0" \
      release="1" \
      summary="MongoDB Enterprise Operator Image" \
      description="MongoDB Enterprise Operator Image"



# Adds up-to-date CA certificates.
RUN apt-get -qq update && \
      apt-get -y -qq install ca-certificates curl && \
      apt-get upgrade -y -qq && \
      apt-get dist-upgrade -y -qq && \
      rm -rf /var/lib/apt/lists/*




COPY --from=base /data/mongodb-enterprise-operator /usr/local/bin/mongodb-enterprise-operator
COPY --from=base /data/version_manifest.json /var/lib/mongodb-enterprise-operator/version_manifest.json
COPY --from=base /data/licenses /licenses/
RUN chmod a+r /var/lib/mongodb-enterprise-operator/version_manifest.json

USER 2000



ENTRYPOINT exec /usr/local/bin/mongodb-enterprise-operator


