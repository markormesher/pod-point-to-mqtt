FROM docker.io/golang:1.26.2@sha256:1e598ea5752ae26c093b746fd73c5095af97d6f2d679c43e83e0eac484a33dc3 AS builder
WORKDIR /app

ARG CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal

RUN go build -o ./build/main ./cmd/...

# ---

FROM ghcr.io/markormesher/scratch:v0.4.17@sha256:5bd7dc42149c5886bca329a551afa544e6336adc3de471d0be7b0f1a9d4638f7
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
