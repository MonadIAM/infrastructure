ARG KAFKA_IMAGE=apache/kafka:3.9.0
ARG SEAWEEDFS_IMAGE=chrislusf/seaweedfs:4.48

FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY cli ./cli
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/infra .

FROM ${KAFKA_IMAGE} AS kafka-runtime
COPY --from=build /out/infra /usr/local/bin/infra
ENTRYPOINT ["/usr/local/bin/infra"]
CMD ["kafka", "start", "--env-file="]

FROM ${SEAWEEDFS_IMAGE} AS seaweedfs-runtime
COPY --from=build /out/infra /usr/local/bin/infra
ENTRYPOINT ["/usr/local/bin/infra"]

FROM scratch AS cli
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/infra /usr/local/bin/infra
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/infra"]
