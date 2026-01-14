# SUP (Signal Unified Push)

Privacy-preserving push notifications using Signal as transport.

## What is SUP?

SUP is a UnifiedPush distributor that routes push notifications through Signal, allowing you to receive app notifications without exposing unique network fingerprints to your ISP or network observers. All notification traffic appears as regular Signal messages.

## Architecture

- **Server** (Bun/TypeScript): Receives UnifiedPush webhooks and forwards them via signal-cli to Signal groups
- **Android App** (Kotlin): Monitors Signal notifications, extracts payloads, and wakes target apps

## Why?

Traditional push notification systems (ntfy, FCM) require persistent WebSocket connections or polling to specific servers, creating unique network fingerprints. SUP blends your notification traffic with regular Signal usage for better privacy.

## Setup

### Prerequisites

- **Docker** (recommended) - includes Java 25 JRE
- **Or for local development:**
  - Bun 1.3+
  - Java 21+ JRE (signal-cli is Java bytecode, not native)

### Quick Start with Docker

```bash
docker run -d -p 8080:8080 -v sup-data:/root/.local/share/signal-cli ghcr.io/lone-cloud/sup:latest
```

Visit `http://localhost:8080/link` to link your Signal account via QR code.

> **Note:** The `-v sup-data:...` persists your Signal account data. Without it, you'll need to re-link on every restart.

**Optional: Add API key for production:**

```bash
docker run -d -p 8080:8080 -e API_KEY=your-secret -v sup-data:/root/.local/share/signal-cli ghcr.io/lone-cloud/sup:latest
```

### Alternative: Docker Compose

```bash
# Clone the repo
git clone https://github.com/lone-cloud/sup.git
cd sup

# Optional: Create .env file for API key
echo "API_KEY=your-secret" > .env

# Run
docker-compose up -d
```

Visit `http://localhost:8080/link` to link your Signal account.

### Development

```bash
bun install
bun dev
```

Visit `http://localhost:8080/link` to link your Signal account.

## API Endpoints

### UnifiedPush Protocol

- `POST /up/{app_id}` - Register new endpoint
- `DELETE /up/{app_id}` - Unregister endpoint
- `GET /up` - Discovery endpoint
- `POST /_matrix/push/v1/notify/{endpoint_id}` - Push notification

### Management

- `GET /health` - Health check
- `GET /endpoints` - List registered endpoints

## How It Works

1. Android app registers with server via `/up/{app_id}`
2. Server creates a Signal group for the app
3. Server returns UnifiedPush endpoint URL
4. App shares endpoint with notification provider
5. Provider sends notifications to endpoint
6. Server forwards to Signal group
7. Android app monitors Signal, extracts payloads, wakes apps

## Android App

Download the latest APK from [GitHub Releases](https://github.com/lone-cloud/sup/releases).

**Install via Obtainium:** [obtainium://add/https://github.com/lone-cloud/sup](obtainium://add/https://github.com/lone-cloud/sup)

**Certificate Fingerprint for Obtainium verification:**

```
0D:3C:99:15:0E:12:1A:DE:0D:AE:05:CB:16:46:5E:65:31:56:DC:D6:98:87:59:4E:79:B1:0D:AE:1E:56:F2:E8
```

Verify this fingerprint when installing via Obtainium to ensure authenticity.
