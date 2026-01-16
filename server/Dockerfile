FROM oven/bun:1.1.42-alpine AS builder

WORKDIR /app

COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile

COPY . .

RUN bun build --compile src/server.ts --outfile sup-server

FROM alpine:3.21

RUN apk add --no-cache openjdk25-jre

COPY --from=builder /app/sup-server /usr/local/bin/sup-server
COPY --from=builder /app/signal-cli /usr/local/bin/signal-cli

EXPOSE 8080

CMD ["sup-server"]
