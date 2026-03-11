FROM docker.io/golang:1.26.1@sha256:cdebbd553e5ed852386e9772e429031467fa44ca3a06735e6beb005d615e623d AS builder
WORKDIR /app

ARG CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal

RUN go build -o ./build/main ./cmd/...

# ---

FROM ghcr.io/markormesher/scratch:v0.4.14@sha256:6212e9b38cc40ad4520f39a4043ac3f7bd12438449356311b4fa327cabce7db0
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
