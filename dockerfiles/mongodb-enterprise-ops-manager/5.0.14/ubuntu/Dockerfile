ARG imagebase
FROM ${imagebase} as base

FROM ubuntu:20.04


LABEL name="MongoDB Enterprise Ops Manager" \
  maintainer="support@mongodb.com" \
  vendor="MongoDB" \
  version="5.0.14" \
  release="1" \
  summary="MongoDB Enterprise Ops Manager Image" \
  description="MongoDB Enterprise Ops Manager"


ENV MMS_HOME /mongodb-ops-manager
ENV MMS_PROP_FILE ${MMS_HOME}/conf/conf-mms.properties
ENV MMS_CONF_FILE ${MMS_HOME}/conf/mms.conf
ENV MMS_LOG_DIR ${MMS_HOME}/logs
ENV MMS_TMP_DIR ${MMS_HOME}/tmp

EXPOSE 8080

# OpsManager docker image needs to have the MongoDB dependencies because the
# backup daemon is running its database locally

RUN apt-get -qq update \
    && apt-get -y -qq install \
        apt-utils \
        ca-certificates \
        curl \
        libsasl2-2 \
        net-tools \
        netcat \
        procps \
        libgssapi-krb5-2 \
        libkrb5-dbg \
        libldap-2.4-2 \
        libpcap0.8 \
        libpci3 \
        libwrap0 \
        libcurl4 \
        liblzma5 \
        libsasl2-modules \
        libsasl2-modules-gssapi-mit\
        openssl \
        snmp \
    && apt-get upgrade -y -qq \
    && apt-get dist-upgrade -y -qq \
    && rm -rf /var/lib/apt/lists/*



COPY --from=base /data/licenses /licenses/



RUN curl --fail -L -o ops_manager.tar.gz https://downloads.mongodb.com/on-prem-mms/tar/mongodb-mms-5.0.14.100.20220802T1010Z-1.x86_64.tar.gz \
  && tar -xzf ops_manager.tar.gz \
  && rm ops_manager.tar.gz \
  && mv mongodb-mms* "${MMS_HOME}"


# permissions
RUN chmod -R 0777 "${MMS_LOG_DIR}" \
  && chmod -R 0777 "${MMS_TMP_DIR}" \
  && chmod -R 0775 "${MMS_HOME}/conf" \
  && chmod -R 0775 "${MMS_HOME}/jdk" \
  && mkdir "${MMS_HOME}/mongodb-releases/" \
  && chmod -R 0775 "${MMS_HOME}/mongodb-releases" \
  && chmod -R 0777 "${MMS_CONF_FILE}" \
  && chmod -R 0777 "${MMS_PROP_FILE}"

# The "${MMS_HOME}/conf" will be populated by the docker-entry-point.sh.
# For now we need to move into the templates directory.
RUN cp -r "${MMS_HOME}/conf" "${MMS_HOME}/conf-template"

USER 2000

# operator to change the entrypoint to: /mongodb-ops-manager/bin/mongodb-mms start_mms (or a wrapper around this)
ENTRYPOINT [ "sleep infinity" ]


