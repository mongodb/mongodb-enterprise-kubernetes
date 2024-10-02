ARG imagebase
FROM ${imagebase} as base

FROM registry.access.redhat.com/ubi8/ubi-minimal


LABEL name="MongoDB Enterprise Ops Manager" \
  maintainer="support@mongodb.com" \
  vendor="MongoDB" \
  version="7.0.11" \
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

RUN microdnf install --disableplugin=subscription-manager -y \
  cyrus-sasl \
  cyrus-sasl-gssapi \
  cyrus-sasl-plain \
  krb5-libs \
  libcurl \
  libpcap \
  lm_sensors-libs \
  net-snmp \
  net-snmp-agent-libs \
  openldap \
  openssl \
  tar \
  rpm-libs \
  net-tools \
  procps-ng \
  ncurses


COPY --from=base /data/licenses /licenses/

COPY --from=base /data/scripts /opt/scripts



RUN curl --fail -L -o ops_manager.tar.gz https://downloads.mongodb.com/on-prem-mms/tar/mongodb-mms-7.0.11.500.20240830T1718Z.tar.gz \
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


