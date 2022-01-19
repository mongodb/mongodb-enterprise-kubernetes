ARG imagebase
FROM ${imagebase} as base

FROM busybox

ARG version
LABEL name="MongoDB Enterprise Init Database" \
      version="mongodb-enterprise-init-database-${version}" \
      summary="MongoDB Enterprise Database Init Image" \
      description="Startup Scripts for MongoDB Enterprise Database" \
      release="1" \
      vendor="MongoDB" \
      maintainer="support@mongodb.com"

COPY --from=base /data/readinessprobe /probes/readinessprobe
COPY --from=base /data/probe.sh /probes/probe.sh
COPY --from=base /data/scripts/ /scripts/
COPY --from=base /data/licenses /licenses/


COPY --from=base /data/mongodb_tools_ubuntu.tgz /tools/mongodb_tools.tgz


RUN tar xfz /tools/mongodb_tools.tgz --directory /tools \
    && rm /tools/mongodb_tools.tgz

USER 2000
ENTRYPOINT [ "/bin/cp", "-f", "-r", "/scripts/agent-launcher.sh", "/scripts/agent-launcher-lib.sh", "/probes/readinessprobe", "/probes/probe.sh", "/tools", "/opt/scripts/" ]

