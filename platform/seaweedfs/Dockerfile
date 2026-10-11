FROM golang:1.27.2-alpine3.23 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY cli ./cli
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /infra .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /infra /infra
USER 65532:65532
ENTRYPOINT ["/infra"]
