<div align="center">

<img src="assets/sup.webp" alt="SUP Icon" width="80" height="80" />

# SUP

**SUP (Signal Unified Push) is a privacy-preserving push notifications using Signal as transport**


[Setup](#setup) • [Architecture](#architecture)

</div>

<!-- markdownlint-enable MD033 -->

SUP is a UnifiedPush distributor that routes push notifications through Signal, allowing you to receive app notifications without exposing unique network fingerprints to your ISP or network observers. All notification traffic appears as regular Signal messages.

## Why?

Traditional push notification systems (ntfy, FCM) require persistent WebSocket connections or polling to specific servers, creating unique network fingerprints. SUP blends your notification traffic with regular Signal usage for better privacy.

## Setup

### 1. Install Android App

Download the latest APK from [GitHub Releases](https://github.com/lone-cloud/sup/releases).

**Certificate Fingerprint:**

```text
0D:3C:99:15:0E:12:1A:DE:0D:AE:05:CB:16:46:5E:65:31:56:DC:D6:98:87:59:4E:79:B1:0D:AE:1E:56:F2:E8
```

### 2. Start SUP Server with Docker Compose on a self-hosted server

**Without ProtonMail** (just UnifiedPush):

```bash
# Download docker-compose.yml
curl -L -O https://raw.githubusercontent.com/lone-cloud/sup/master/docker-compose.yml

# Download .env.example (optional)
curl -L -O https://raw.githubusercontent.com/lone-cloud/sup/master/.env.example

# Configure (optional)
cp .env.example .env
nano .env

# Start SUP server
docker compose up -d

# Link your Signal account (one-time setup)
# Visit http://localhost:8080/link and scan QR code with Signal app
```

### 3. ProtonMail Integration (Optional)

> **Note:** The default ProtonMail Bridge image uses `shenxn/protonmail-bridge:build` which compiles from source and supports multiple architectures. For x86_64 systems, you can use `shenxn/protonmail-bridge:latest` (pre-built binary, smaller and faster). For ARM devices (Raspberry Pi), stick with `:build`.

To receive ProtonMail notifications via Signal:

1. **Initialize ProtonMail Bridge** (one-time setup):

   ```bash
   docker compose run --rm protonmail-bridge init
   ```
  
2. **Login to ProtonMail**:
   - At the `>>>` prompt, run: `login`
   - Enter your ProtonMail email
   - Enter your ProtonMail password
   - Enter your 2FA code
   - Wait (potentially a long time) for ProtonMail Bridge to sync emails

3. **Get IMAP credentials**:
   - Run: `info`
   - Copy the Username and Password shown
   - Run: `exit` to quit

4. **Add credentials to .env**:

   ```bash
   # Add these to your .env file
   BRIDGE_IMAP_USERNAME=bridge-username-from-info-command
   BRIDGE_IMAP_PASSWORD=bridge-generated-password-from-info-command
   ```

5. **Start all services with ProtonMail**:

   ```bash
   docker compose --profile protonmail up -d
   ```

Your phone will now receive Signal notifications when ProtonMail receives new emails.

### Development

For local development, install Bun and signal-cli:

```bash
# Install Bun (use your package manager and this is a backup)
curl -fsSL https://bun.sh/install | bash

git clone https://github.com/lone-cloud/sup.git
cd sup

bun install
```

Then build and run with docker-compose.dev.yml:

```bash
docker compose -f docker-compose.dev.yml --profile protonmail up -d
```

Or run services directly with Bun:

```bash
bun install
bun --filter sup-server dev
```

## Architecture

![SUP Architecture](assets/SUP%20Architecture.webp)

SUP consists of two services that **MUST run together on the same machine**:

- **sup-server** (Bun): Receives webhooks, sends Signal messages via signal-cli. Optional: monitors ProtonMail IMAP
- **protonmail-bridge** (Official Proton, optional): Decrypts ProtonMail emails, runs local IMAP server

All services communicate over a private Docker network with no external exposure except Signal protocol. **Separating these services across multiple machines would expose plaintext IMAP traffic and compromise security.**

**Android App** (Kotlin): Monitors Signal notifications, extracts UnifiedPush payloads, delivers to apps
