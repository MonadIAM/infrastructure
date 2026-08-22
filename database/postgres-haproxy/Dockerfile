ARG HAPROXY_BASE_IMAGE=haproxy:2.9-alpine

FROM hashicorp/consul-template:0.39 AS consul-template

FROM ${HAPROXY_BASE_IMAGE}

USER root
COPY --from=consul-template /bin/consul-template /usr/local/bin/consul-template
COPY haproxy.ctmpl /etc/haproxy-templates/haproxy.ctmpl
RUN mkdir -p /usr/local/etc/haproxy \
    && chown haproxy:haproxy /usr/local/etc/haproxy \
    && chmod 0755 /etc/haproxy-templates \
    && chmod 0644 /etc/haproxy-templates/haproxy.ctmpl
USER haproxy

ENTRYPOINT ["consul-template"]
