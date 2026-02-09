FROM golang:1.25-alpine3.23 AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-w -s -X main.version=$(cat VERSION 2>/dev/null || echo dev) -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" \
    -o prism .

FROM debian:trixie-slim

ARG TARGETARCH

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates wget curl && \
    curl -L -o signal-cli.gz \
        https://media.projektzentrisch.de/temp/signal-cli/signal-cli_ubuntu2004_${TARGETARCH}.gz && \
    gunzip signal-cli.gz && \
    mv signal-cli /usr/local/bin/signal-cli && \
    chmod +x /usr/local/bin/signal-cli && \
    apt-get remove -y curl && \
    apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /build/prism .

RUN useradd -m -u 1000 prism && \
    mkdir -p /app/data && \
    chown -R prism:prism /app

USER prism

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./prism"]
