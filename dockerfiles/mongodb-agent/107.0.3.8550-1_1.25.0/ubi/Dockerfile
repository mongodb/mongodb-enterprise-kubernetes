ARG imagebase
FROM ${imagebase} as base

FROM registry.access.redhat.com/ubi8/ubi-minimal

ARG version

LABEL name="MongoDB Agent" \
      version="${agent_version}" \
      summary="MongoDB Agent" \
      description="MongoDB Agent" \
      vendor="MongoDB" \
      release="1" \
      maintainer="support@mongodb.com"

COPY --from=base /data/probe.sh /opt/scripts/probe.sh
COPY --from=base /data/readinessprobe /opt/scripts/readinessprobe
COPY --from=base /data/version-upgrade-hook /opt/scripts/version-upgrade-hook
COPY --from=base /data/agent-launcher-lib.sh /opt/scripts/agent-launcher-lib.sh
COPY --from=base /data/agent-launcher.sh /opt/scripts/agent-launcher.sh
COPY --from=base /data/LICENSE /LICENSE

RUN microdnf install -y --disableplugin=subscription-manager --setopt=install_weak_deps=0 nss_wrapper
# Copy-pasted from https://www.mongodb.com/docs/manual/tutorial/install-mongodb-enterprise-on-red-hat-tarball/
RUN microdnf install -y --disableplugin=subscription-manager --setopt=install_weak_deps=0 \
 cyrus-sasl cyrus-sasl-gssapi cyrus-sasl-plain krb5-libs libcurl openldap openssl xz-libs
# Dependencies for the Agent
RUN microdnf install -y --disableplugin=subscription-manager  --setopt=install_weak_deps=0 \
        net-snmp \
        net-snmp-agent-libs
RUN microdnf install -y --disableplugin=subscription-manager curl \
    hostname tar gzip procps jq \
    && microdnf upgrade -y  \
    && rm -rf /var/lib/apt/lists/*


COPY --from=base /data/mongodb_tools_ubi.tgz /tools/mongodb_tools.tgz
COPY --from=base /data/mongodb_agent_ubi.tgz /agent/mongodb_agent.tgz

RUN tar xfz /tools/mongodb_tools.tgz
RUN mv mongodb-database-tools-*/bin/* /tools
RUN chmod +x /tools/*
RUN rm /tools/mongodb_tools.tgz
RUN rm -rf /mongodb-database-tools-*

RUN tar xfz /agent/mongodb_agent.tgz
RUN mv mongodb-mms-automation-agent-*/mongodb-mms-automation-agent /agent/mongodb-agent
RUN chmod +x /agent/mongodb-agent
RUN rm /agent/mongodb_agent.tgz
RUN rm -rf mongodb-mms-automation-agent-*

RUN mkdir -p /var/lib/automation/config
RUN chmod -R +r /var/lib/automation/config

USER 2000

HEALTHCHECK --timeout=30s CMD ls /opt/scripts/readinessprobe || exit 1