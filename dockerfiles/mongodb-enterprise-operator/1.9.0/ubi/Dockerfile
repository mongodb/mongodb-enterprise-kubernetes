#
# Base Template Dockerfile for Operator Image.
#

ARG imagebase
FROM ${imagebase} as base

FROM registry.access.redhat.com/ubi8/ubi


LABEL name="MongoDB Enterprise Operator" \
      maintainer="support@mongodb.com" \
      vendor="MongoDB" \
      version="1.9.0-23-g9050b474" \
      release="1" \
      summary="MongoDB Enterprise Operator Image" \
      description="MongoDB Enterprise Operator Image"


RUN yum -y --disableplugin=subscription-manager update && \
    yum -y --disableplugin=subscription-manager clean all




COPY --from=base /data/mongodb-enterprise-operator /usr/local/bin/mongodb-enterprise-operator
COPY --from=base /data/version_manifest.json /var/lib/mongodb-enterprise-operator/version_manifest.json
COPY --from=base /data/licenses /licenses/
RUN chmod a+r /var/lib/mongodb-enterprise-operator/version_manifest.json

USER 2000

ENTRYPOINT exec /usr/local/bin/mongodb-enterprise-operator


