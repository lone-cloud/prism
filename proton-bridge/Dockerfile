FROM oven/bun:1.1.42-alpine AS builder

WORKDIR /app

COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile

COPY . .

RUN bun build --compile src/index.ts --outfile sup-proton-bridge

FROM alpine:3.21

COPY --from=builder /app/sup-proton-bridge /usr/local/bin/sup-proton-bridge

CMD ["sup-proton-bridge"]
