FROM docker.io/golang:1.26.0@sha256:c83e68f3ebb6943a2904fa66348867d108119890a2c6a2e6f07b38d0eb6c25c5 AS builder
WORKDIR /app

ARG CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal

RUN go build -o ./build/main ./cmd/...

# ---

FROM ghcr.io/markormesher/scratch:v0.4.13@sha256:4322b7982b9bd492ba1f69f7abf5cfe3061f2c9c20e8970fa28ebacc3964df89
WORKDIR /app

COPY --from=builder /app/build/main /usr/local/bin/pod-point-to-mqtt

CMD ["/usr/local/bin/pod-point-to-mqtt"]

LABEL image.name=markormesher/pod-point-to-mqtt
LABEL image.registry=ghcr.io
LABEL org.opencontainers.image.description=""
LABEL org.opencontainers.image.documentation=""
LABEL org.opencontainers.image.title="pod-point-to-mqtt"
LABEL org.opencontainers.image.url="https://github.com/markormesher/pod-point-to-mqtt"
LABEL org.opencontainers.image.vendor=""
LABEL org.opencontainers.image.version=""
