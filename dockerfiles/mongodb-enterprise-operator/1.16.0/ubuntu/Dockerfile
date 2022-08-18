#
# Base Template Dockerfile for Operator Image.
#

ARG imagebase
FROM ${imagebase} as base

FROM ubuntu:20.04


LABEL name="MongoDB Enterprise Operator" \
      maintainer="support@mongodb.com" \
      vendor="MongoDB" \
      version="1.16.0" \
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
COPY --from=base /data/om_version_mapping.json /usr/local/om_version_mapping.json
COPY --from=base /data/licenses /licenses/

USER 2000



ENTRYPOINT exec /usr/local/bin/mongodb-enterprise-operator


