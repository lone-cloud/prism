FROM oven/bun:1.3.7-debian AS builder

WORKDIR /app

COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile

COPY server ./server

RUN bun build --compile server/index.ts --outfile sup

FROM debian:13.3-slim

ARG SIGNAL_CLI_VERSION=${SIGNAL_CLI_VERSION:-0.13.23}

RUN apt-get update && apt-get install -y --no-install-recommends \
    openjdk-21-jre-headless \
    curl \
    ca-certificates \
    zip \
    && rm -rf /var/lib/apt/lists/*

RUN curl -L https://github.com/AsamK/signal-cli/releases/download/v${SIGNAL_CLI_VERSION}/signal-cli-${SIGNAL_CLI_VERSION}.tar.gz | tar xz -C /tmp \
    && mv /tmp/signal-cli-${SIGNAL_CLI_VERSION} /usr/local/signal-cli \
    && chmod +x /usr/local/signal-cli/bin/signal-cli

RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "aarch64" ]; then \
        echo "Installing ARM64 native library for signal-cli..." && \
        LIBSIGNAL_VERSION=$(ls /usr/local/signal-cli/lib/libsignal-client-*.jar | sed 's/.*libsignal-client-\(.*\)\.jar/\1/') && \
        curl -L https://github.com/exquo/signal-libs-build/releases/download/libsignal_v${LIBSIGNAL_VERSION}/libsignal_jni.so-v${LIBSIGNAL_VERSION}-aarch64-unknown-linux-gnu.tar.gz | tar xz -C /tmp && \
        mv /tmp/libsignal_jni.so /tmp/libsignal_jni_aarch64.so && \
        cd /tmp && zip -u /usr/local/signal-cli/lib/libsignal-client-*.jar libsignal_jni_aarch64.so && \
        rm /tmp/libsignal_jni_aarch64.so; \
    fi

COPY --from=builder /app/sup /usr/local/bin/sup
COPY --from=builder /app/server/public /public

ENV PATH="/usr/local/signal-cli/bin:${PATH}"
ENV LD_LIBRARY_PATH="/usr/local/signal-cli/lib:${LD_LIBRARY_PATH}"

EXPOSE 8080

CMD ["sup"]
