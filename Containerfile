FROM docker.io/golang:1.25.0@sha256:10a15b9d650c559eff6cb070f3177f1e2fc067cd7412e5ca97c9cb8167a924b7 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal

RUN go build -o ./build/main ./cmd/...

# ---

FROM gcr.io/distroless/base-debian12@sha256:4f6e739881403e7d50f52a4e574c4e3c88266031fd555303ee2f1ba262523d6a
WORKDIR /app

LABEL image.registry=ghcr.io
LABEL image.name=markormesher/pod-point-to-mqtt

COPY --from=builder /app/build/main /usr/local/bin/pod-point-to-mqtt

CMD ["/usr/local/bin/pod-point-to-mqtt"]
