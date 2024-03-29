ARG imagebase
FROM ${imagebase} as base

FROM registry.access.redhat.com/ubi7/ubi

RUN yum install -y  --disableplugin=subscription-manager -q curl \
    hostname nss_wrapper --exclude perl-IO-Socket-SSL procps \
    && yum upgrade -y -q \
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