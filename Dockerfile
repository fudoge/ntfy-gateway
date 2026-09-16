# syntax=docker/dockerfile:1

FROM golang:1.26.8-alpine3.24 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/ntfy-gateway ./cmd/ntfy-gateway

FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=build --chown=nonroot:nonroot /out/ntfy-gateway /app/ntfy-gateway

USER nonroot:nonroot

EXPOSE 8080

ENV CONFIG_PATH=/etc/ntfy-gateway/config.yaml

ENTRYPOINT ["/app/ntfy-gateway"]
