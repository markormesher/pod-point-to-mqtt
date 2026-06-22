FROM docker.io/golang:1.26.4@sha256:792443b89f65105abba56b9bd5e97f680a80074ac62fc844a584212f8c8102c3 AS builder
WORKDIR /app

ARG CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal

RUN go build -o ./build/main ./cmd/...

# ---

FROM ghcr.io/markormesher/scratch:v0.4.22@sha256:2d472a373e6864cf79007158f8dfd4f67b3ff68e7a40350584c447ae8aa0598e
WORKDIR /app

COPY --from=builder /app/build/main /usr/local/bin/pod-point-to-mqtt

CMD ["/usr/local/bin/pod-point-to-mqtt"]

LABEL image.name=markormesher/pod-point-to-mqtt
LABEL image.registry=ghcr.io
LABEL org.opencontainers.image.description=""
LABEL org.opencontainers.image.documentation=""
LABEL org.opencontainers.image.title="pod-point-to-mqtt"
LABEL org.opencontainers.image.url=""
LABEL org.opencontainers.image.vendor=""
LABEL org.opencontainers.image.version=""
