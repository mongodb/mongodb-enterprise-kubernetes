ARG imagebase
FROM ${imagebase} as base

FROM ubuntu:16.04




LABEL name="MongoDB Enterprise AppDB" \
      version="10.2.15.5958-1_4.2.11-ent" \
      summary="MongoDB Enterprise AppDB" \
      description="MongoDB Enterprise AppDB" \
      vendor="MongoDB" \
      release="1" \
      maintainer="support@mongodb.com"






ENV MMS_HOME /mongodb-automation
ENV MMS_LOG_DIR /var/log/mongodb-mms-automation







# Download and extract the MongoDB archive and put it where the automation agent
# expects to find it. The Mongodb agent will download MongoDB from the Internet
# only if the version does not match the version we have put inside the image.

COPY --from=base /data/mongodb_server_ubuntu.tgz /tmp
RUN mkdir -p /var/lib/mongodb-mms-automation/downloads \
    && tar -xzf "/tmp/mongodb_server_ubuntu.tgz" --directory "/var/lib/mongodb-mms-automation/downloads" \
    && rm "/tmp/mongodb_server_ubuntu.tgz" \
    && chmod -R 0775 "/var/lib/mongodb-mms-automation/downloads"

COPY --from=base /data/mongodb_agent_linux.tgz /tmp
RUN mkdir -p ${MMS_HOME}/files/ \
    && tar -xzf /tmp/mongodb_agent_linux.tgz --directory / \
    && mv /mongodb-mms-automation-agent-*/mongodb-mms-automation-agent "${MMS_HOME}/files/" \
    && chmod +x "${MMS_HOME}/files/mongodb-mms-automation-agent" \
    && rm -rf /tmp/mongodb_agent_linux.tgz "/mongodb-mms-automation-agent-*/"

RUN echo "10.2.15.5958-1" > ${MMS_HOME}/files/agent-version





RUN apt-get -qq update \
        && apt-get -y -qq install \
        curl \
        jq \
        libcurl3 \
        libgssapi-krb5-2 \
        libkrb5-dbg \
        libldap-2.4-2 \
        libpcap0.8 \
        libsasl2-2 \
        lsb-release \
        openssl \
        snmp \
        libnss-wrapper \
        && apt-get upgrade -y -qq \
        && apt-get dist-upgrade -y -qq \
        && rm -rf /var/lib/apt/lists/*


# Set the required perms
RUN    mkdir -p "${MMS_LOG_DIR}" \
        && chmod 0775 "${MMS_LOG_DIR}" \
        && mkdir -p /var/lib/mongodb-mms-automation \
        && chmod 0775 /var/lib/mongodb-mms-automation \
        && mkdir -p /data \
        && chmod 0775 /data \
        && mkdir -p /journal \
        && chmod 0775 /journal \
        && mkdir -p "${MMS_HOME}" \
        && chmod -R 0775 "${MMS_HOME}"




# USER needs to be set for this image to pass RedHat verification. Some customers have these requirements as well
# It does not matter what number it is, as long as it is set to something.
# However, OpenShift will run the container as a random user,
# and the number in this configuration is not relevant.
USER 2000


# The docker image doesn't have any scripts so by default does nothing
# The script will be copied in runtime from init containers and the operator is expected
# to override the COMMAND
ENTRYPOINT ["sleep infinity"]


COPY --from=base /data/licenses/mongodb-enterprise-database /licenses/mongodb-enterprise-database


