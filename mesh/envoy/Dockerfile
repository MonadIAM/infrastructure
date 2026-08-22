ARG CONSUL_IMAGE=hashicorp/consul:1.20
ARG ENVOY_IMAGE=hashicorp/envoy:1.33.2

FROM ${CONSUL_IMAGE} AS consul
FROM ${ENVOY_IMAGE}

COPY --from=consul /bin/consul /bin/consul

ENTRYPOINT ["consul"]
