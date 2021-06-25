ARG imagebase
FROM ${imagebase} as base

FROM busybox

LABEL name="MongoDB Enterprise Ops Manager Init" \
      maintainer="support@mongodb.com" \
      vendor="MongoDB" \
      version="mongodb-enterprise-init-ops-manager-1.0.3" \
      release="1" \
      summary="MongoDB Enterprise Ops Manager Init Image" \
      description="Startup Scripts for MongoDB Enterprise Ops Manager"


COPY --from=base /data/scripts /scripts
COPY --from=base /data/licenses /licenses

USER 2000
ENTRYPOINT [ "/bin/cp", "-f", "/scripts/docker-entry-point.sh", "/scripts/mmsconfiguration", "/opt/scripts/" ]


