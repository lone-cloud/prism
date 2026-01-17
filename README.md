# SUP (Signal Unified Push)

Privacy-preserving push notifications using Signal as transport.

## What is SUP?

SUP is a UnifiedPush distributor that routes push notifications through Signal, allowing you to receive app notifications without exposing unique network fingerprints to your ISP or network observers. All notification traffic appears as regular Signal messages.

## Architecture

SUP consists of three services that **MUST run together on the same machine**:

- **sup-server** (Bun/TypeScript): Receives webhooks, sends Signal messages via signal-cli
- **protonmail-bridge** (Official Proton): Decrypts ProtonMail emails, runs local IMAP server
- **sup-proton-bridge** (Custom): Monitors IMAP, forwards to sup-server

All services communicate over a private Docker network with no external exposure except Signal protocol. **Separating these services across multiple machines would expose plaintext IMAP traffic and compromise security.**

**Android App** (Kotlin): Monitors Signal notifications, extracts UnifiedPush payloads, delivers to apps

## Why?

Traditional push notification systems (ntfy, FCM) require persistent WebSocket connections or polling to specific servers, creating unique network fingerprints. SUP blends your notification traffic with regular Signal usage for better privacy.

## Setup

**⚠️ DOCKER COMPOSE REQUIRED**: The services must be deployed together using `docker compose`. Running individual Dockerfiles separately is not supported and will compromise security.

### Prerequisites

#### Installing Docker on Arch Linux

```bash
# Install Docker and Compose plugin
sudo pacman -S docker docker-buildx

# Start Docker service
sudo systemctl start docker
sudo systemctl enable docker

# Add your user to docker group (logout/login required)
sudo usermod -aG docker $USER
```

After adding yourself to the docker group, **logout and login** for it to take effect.

### Quick Start with Docker Compose

**Without ProtonMail** (just UnifiedPush):

```bash
# Download docker-compose.yml
curl -L -O https://raw.githubusercontent.com/lone-cloud/sup/master/docker-compose.yml

# Create .env file
cat > .env << 'EOF'
# Optional: API key for remote access
# Set this to protect your server when accessing it from outside your home network
# (e.g., registering UnifiedPush apps while away from home)
# Default: unset (no authentication required)
API_KEY=your-random-secret-key-here

# Optional: Enable verbose logging
# Default: false
VERBOSE=false
EOF

# Start SUP server
docker compose up -d

# Link your Signal account (one-time setup)
# Visit http://localhost:8080/link and scan QR code with Signal app
```

### ProtonMail Integration (Optional)

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
   BRIDGE_IMAP_USERNAME=your-email@proton.me
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

**Certificate Fingerprint:**

```text
0D:3C:99:15:0E:12:1A:DE:0D:AE:05:CB:16:46:5E:65:31:56:DC:D6:98:87:59:4E:79:B1:0D:AE:1E:56:F2:E8
```
