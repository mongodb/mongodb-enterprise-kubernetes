ARG imagebase
FROM ${imagebase} as base

FROM registry.access.redhat.com/ubi8/ubi-minimal

ARG agent_version

LABEL name="MongoDB Agent" \
      version="${agent_version}" \
      summary="MongoDB Agent" \
      description="MongoDB Agent" \
      vendor="MongoDB" \
      release="1" \
      maintainer="support@mongodb.com"

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

RUN mkdir -p /agent \
    && mkdir -p /var/lib/mongodb-mms-automation \
      && mkdir -p /var/log/mongodb-mms-automation/ \
      && chmod -R +wr /var/log/mongodb-mms-automation/ \
      # ensure that the agent user can write the logs in OpenShift
      && touch /var/log/mongodb-mms-automation/readiness.log \
      && chmod ugo+rw /var/log/mongodb-mms-automation/readiness.log


COPY --from=base /data/mongodb-agent.tar.gz /agent
COPY --from=base /data/mongodb-tools.tgz /agent
COPY --from=base /data/LICENSE /licenses/LICENSE

RUN tar xfz /agent/mongodb-agent.tar.gz \
    && mv mongodb-mms-automation-agent-*/mongodb-mms-automation-agent /agent/mongodb-agent \
    && chmod +x /agent/mongodb-agent \
    && mkdir -p /var/lib/automation/config \
    && chmod -R +r /var/lib/automation/config \
    && rm /agent/mongodb-agent.tar.gz \
    && rm -r mongodb-mms-automation-agent-*

RUN tar xfz /agent/mongodb-tools.tgz --directory /var/lib/mongodb-mms-automation/ && rm /agent/mongodb-tools.tgz

USER 2000
CMD ["/agent/mongodb-agent", "-cluster=/var/lib/automation/config/automation-config.json"]