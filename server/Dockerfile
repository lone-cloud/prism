FROM oven/bun:1.3.6-alpine AS builder

WORKDIR /app

COPY server/package.json server/bun.lock* ./
RUN bun install --frozen-lockfile

COPY server .

RUN bun build --compile index.ts --outfile sup-server

FROM alpine:3.23

ARG SIGNAL_CLI_VERSION=0.13.22

RUN apk add --no-cache openjdk21-jre curl libstdc++ libgcc gcompat

RUN curl -L https://github.com/AsamK/signal-cli/releases/download/v${SIGNAL_CLI_VERSION}/signal-cli-${SIGNAL_CLI_VERSION}.tar.gz | tar xz -C /tmp \
    && mv /tmp/signal-cli-${SIGNAL_CLI_VERSION} /usr/local/signal-cli \
    && chmod +x /usr/local/signal-cli/bin/signal-cli

RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "aarch64" ]; then \
        echo "Installing ARM64 native library for signal-cli..." && \
        apk add --no-cache unzip zip && \
        curl -L -o /tmp/libsignal_jni.so https://github.com/exquo/signal-libs-build/releases/latest/download/libsignal_jni_linux_aarch64.so && \
        unzip -q /usr/local/signal-cli/lib/libsignal-client-*.jar -d /tmp/jar && \
        rm /usr/local/signal-cli/lib/libsignal-client-*.jar && \
        cp /tmp/libsignal_jni.so /tmp/jar/libsignal_jni_aarch64.so && \
        cd /tmp/jar && zip -qr /usr/local/signal-cli/lib/libsignal-client.jar . && \
        cd - && rm -rf /tmp/jar /tmp/libsignal_jni.so && \
        apk del unzip zip; \
    fi

COPY --from=builder /app/sup-server /usr/local/bin/sup-server

ENV PATH="/usr/local/signal-cli/bin:${PATH}"
ENV LD_LIBRARY_PATH="/usr/local/signal-cli/lib:${LD_LIBRARY_PATH}"

EXPOSE 8080

CMD ["sup-server"]
