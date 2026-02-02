FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s -X main.version=$(git describe --tags --always) -X main.commit=$(git rev-parse --short HEAD)" \
    -o prism .

FROM alpine:latest

RUN apk --no-cache add ca-certificates signal-cli openjdk21-jre

WORKDIR /app

COPY --from=builder /build/prism .
COPY public ./public

RUN adduser -D -u 1000 prism && \
    mkdir -p /var/run/signal-cli /app/data && \
    chown -R prism:prism /app /var/run/signal-cli

USER prism

ENV SIGNAL_CLI_BINARY=signal-cli
ENV SIGNAL_CLI_SOCKET=/var/run/signal-cli/socket

EXPOSE 8080

CMD ["./prism"]
