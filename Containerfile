FROM docker.io/golang:1.25.3@sha256:6d4e5e74f47db00f7f24da5f53c1b4198ae46862a47395e30477365458347bf2 AS builder
WORKDIR /app

ARG CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal

RUN go build -o ./build/main ./cmd/...

# ---

FROM ghcr.io/markormesher/scratch:v0.4.5@sha256:702338aef8b7b5427b29079992b05cb912161256edfd36114b52f7f3bbc54a95
WORKDIR /app

LABEL image.registry=ghcr.io
LABEL image.name=markormesher/pod-point-to-mqtt

COPY --from=builder /app/build/main /usr/local/bin/pod-point-to-mqtt

CMD ["/usr/local/bin/pod-point-to-mqtt"]
