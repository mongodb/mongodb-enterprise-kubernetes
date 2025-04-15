ARG imagebase
FROM ${imagebase} as base

FROM registry.access.redhat.com/ubi8/ubi-minimal



LABEL name="MongoDB Enterprise Database" \
      version="1.33.0" \
      summary="MongoDB Enterprise Database Image" \
      description="MongoDB Enterprise Database Image" \
      vendor="MongoDB" \
      release="1" \
      maintainer="support@mongodb.com"





ENV MMS_HOME /mongodb-automation
ENV MMS_LOG_DIR /var/log/mongodb-mms-automation



RUN microdnf update -y && rm -rf /var/cache/yum

# these are the packages needed for the agent
RUN microdnf install -y --disableplugin=subscription-manager --setopt=install_weak_deps=0 nss_wrapper
RUN microdnf install -y --disableplugin=subscription-manager \
        hostname \
        procps


# these are the packages needed for MongoDB
# (https://docs.mongodb.com/manual/tutorial/install-mongodb-enterprise-on-red-hat-tarball/ "RHEL/CentOS 8" tab)
RUN microdnf install -y --disableplugin=subscription-manager \
        cyrus-sasl \
        cyrus-sasl-gssapi \
        cyrus-sasl-plain \
        krb5-libs \
        libcurl \
        lm_sensors-libs \
        net-snmp \
        net-snmp-agent-libs \
        openldap \
        openssl \
        jq \ 
        tar \
        xz-libs \
        findutils



RUN ln -s /usr/lib64/libsasl2.so.3 /usr/lib64/libsasl2.so.2


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


