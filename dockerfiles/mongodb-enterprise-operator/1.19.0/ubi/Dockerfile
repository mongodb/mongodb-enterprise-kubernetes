#
# Base Template Dockerfile for Operator Image.
#

ARG imagebase
FROM ${imagebase} as base

FROM registry.access.redhat.com/ubi8/ubi-minimal


LABEL name="MongoDB Enterprise Operator" \
      maintainer="support@mongodb.com" \
      vendor="MongoDB" \
      version="1.19.0" \
      release="1" \
      summary="MongoDB Enterprise Operator Image" \
      description="MongoDB Enterprise Operator Image"


# Building an UBI-based image: https://red.ht/3n6b9y0
RUN microdnf update \
    --disableplugin=subscription-manager \
    --disablerepo=* --enablerepo=ubi-8-appstream-rpms --enablerepo=ubi-8-baseos-rpms -y \
    && rm -rf /var/cache/yum




COPY --from=base /data/mongodb-enterprise-operator /usr/local/bin/mongodb-enterprise-operator
COPY --from=base /data/om_version_mapping.json /usr/local/om_version_mapping.json
COPY --from=base /data/licenses /licenses/

USER 2000



ENTRYPOINT exec /usr/local/bin/mongodb-enterprise-operator


