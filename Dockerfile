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

FROM alpine:3.23

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/prism .

RUN adduser -D -u 1000 prism && \
    mkdir -p /app/data && \
    chown -R prism:prism /app

USER prism

EXPOSE 8080

CMD ["./prism"]
