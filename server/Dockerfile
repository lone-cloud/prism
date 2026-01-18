FROM oven/bun:1.3.6-alpine AS builder

WORKDIR /app

COPY server/package.json server/bun.lock* ./
RUN bun install --frozen-lockfile

COPY server .

RUN bun build --compile index.ts --outfile sup-server

FROM alpine:3.23

ARG SIGNAL_CLI_VERSION=0.13.22

RUN apk add --no-cache openjdk21-jre curl libstdc++ libgcc

RUN curl -L https://github.com/AsamK/signal-cli/releases/download/v${SIGNAL_CLI_VERSION}/signal-cli-${SIGNAL_CLI_VERSION}.tar.gz | tar xz -C /tmp \
    && mv /tmp/signal-cli-${SIGNAL_CLI_VERSION} /usr/local/signal-cli

COPY --from=builder /app/sup-server /usr/local/bin/sup-server

ENV PATH="/usr/local/signal-cli/bin:${PATH}"

EXPOSE 8080

CMD ["sup-server"]
