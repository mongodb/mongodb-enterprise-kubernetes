ARG imagebase
FROM ${imagebase} as base

FROM ubuntu:16.04




LABEL name="MongoDB Enterprise Database" \
      version="2.0.0" \
      summary="MongoDB Enterprise Database Image" \
      description="MongoDB Enterprise Database Image" \
      vendor="MongoDB" \
      release="1" \
      maintainer="support@mongodb.com"






ENV MMS_HOME /mongodb-automation
ENV MMS_LOG_DIR /var/log/mongodb-mms-automation







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


