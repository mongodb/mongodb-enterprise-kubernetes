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

COPY --from=base /data/readinessprobe /probes/readinessprobe
COPY --from=base /data/probe.sh /probes/probe.sh
COPY --from=base /data/scripts/ /scripts/
COPY --from=base /data/licenses /licenses/

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

RUN tar xfz /tools/mongodb_tools.tgz --directory /tools \
    && rm /tools/mongodb_tools.tgz


RUN tar xfz /agent/mongodb_agent.tgz
RUN ls /agent
RUN mv mongodb-mms-automation-agent-*/mongodb-mms-automation-agent /agent/mongodb-agent
RUN ls /agent
RUN chmod +x /agent/mongodb-agent
RUN mkdir -p /var/lib/automation/config
RUN ls /var/lib/automation/config
RUN chmod -R +r /var/lib/automation/config
RUN ls /var/lib/automation/config
RUN rm /agent/mongodb_agent.tgz
RUN rm -r mongodb-mms-automation-agent-*

USER 2000
ENTRYPOINT [ "/bin/cp", "-f", "-r", "/scripts/agent-launcher.sh", "/scripts/agent-launcher-lib.sh", "/probes/readinessprobe", "/probes/probe.sh", "/tools", "/opt/scripts/" ]

HEALTHCHECK --timeout=30s CMD ls /probes/readinessprobe || exit 1