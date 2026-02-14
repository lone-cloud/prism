FROM golang:1.25-alpine3.23 AS builder

ARG VERSION=dev

WORKDIR /build

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o prism .

FROM debian:trixie-slim

ARG TARGETARCH

COPY signal-cli/signal-cli-${TARGETARCH}.gz /tmp/signal-cli.gz

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates wget && \
    gunzip /tmp/signal-cli.gz && \
    mv /tmp/signal-cli /usr/local/bin/signal-cli && \
    chmod +x /usr/local/bin/signal-cli && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /build/prism .

RUN useradd -m -u 1000 prism && \
    mkdir -p /app/data && \
    mkdir -p /home/prism/.local/share/signal-cli && \
    chown -R prism:prism /app /home/prism

USER prism

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8080}/health || exit 1

CMD ["./prism"]
