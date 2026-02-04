<div align="center">

<img src="assets/prism.webp" alt="Prism Icon" width="120" height="120" />

# Prism

**Self-hosted notification gateway using Signal and WebPush for transport**

[Setup](#setup) • [Real-World Examples](#real-world-examples) • [Architecture](#architecture)

</div>

<!-- markdownlint-enable MD033 -->

Prism is a self-hosted notification gateway that receives HTTP requests and routes them through Signal groups or WebPush apps. Route notifications through Signal to avoid exposing unique network fingerprints, or forward them to your own WebPush apps for custom handling.


## Setup

### 1. Proton Mail Integration

A Proton Mail Bridge is optionally available if you want to receive push notifications for incoming emails.

> **Note:** The default Proton Mail Bridge image uses `shenxn/protonmail-bridge:build` which compiles from source and supports multiple architectures. For x86_64 systems, you can use `shenxn/protonmail-bridge:latest` (pre-built binary, smaller and faster). For ARM devices (Raspberry Pi), stick with `:build`.

To receive Proton Mail notifications via Signal:

1. **Initialize Proton Mail Bridge** (one-time setup):

```bash
# Download docker-compose.yml
curl -L -O https://raw.githubusercontent.com/lone-cloud/prism/master/docker-compose.yml

docker compose run --rm protonmail-bridge init
```

2.**Login to Proton Mail Bridge**:

- At the `>>>` prompt, run: `login`
- Enter your email
- Enter your password
- Enter your 2FA code

3.**Get IMAP credentials**:

- Run: `info`
- Copy the Username and Password shown
- Run: `exit` to quit

4.**Add credentials to .env**:

```bash
# Add these to your .env file
PROTON_IMAP_USERNAME=bridge-username-from-info-command
PROTON_IMAP_PASSWORD=bridge-generated-password-from-info-command
```

5.**Start all services with Proton Mail**:

```bash
docker compose --profile protonmail up -d
```

Your phone will now receive Signal notifications when Proton Mail receives new emails.

Note that the bridge will first need to sync all of your old emails before you can start getting new email notifications which may take a while, but this is a one-time setup.

### 2. Install Prism Server

```bash
# Download docker-compose.yml
curl -L -O https://raw.githubusercontent.com/lone-cloud/prism/master/docker-compose.yml

# Download .env.example (optional)
curl -L -O https://raw.githubusercontent.com/lone-cloud/prism/master/.env.example

# Configure Prism server through environment variables (optional)
cp .env.example .env
nano .env

# Start Prism server
docker compose up -d

```

### 3. Link Your Signal Account

Visit <http://localhost:8080> and link your Signal account (one-time setup):

#### 1. Authenticate with your API_KEY

![Admin login screen](assets/screenshots/1.webp)

#### 2. Scan the QR code from your Signal app

Go to **Settings → Linked Devices → Link New Device** in Signal.

![QR code linking screen](assets/screenshots/2.webp)

#### 3. Verify the setup

Once linked, you'll see the status dashboard:

![Healthy setup with linked account](assets/screenshots/3.webp)

With optional Proton Mail integration:

![Healthy setup with Proton Mail](assets/screenshots/4.webp)

### Development

For local development, install Go and signal-cli:

```bash
git clone https://github.com/lone-cloud/prism.git
cd prism

# Install development tools and signal-cli
make install-tools

# Run locally
make dev
```

Then build and run with docker-compose.dev.yml:

```bash
docker compose --profile protonmail -f docker-compose.dev.yml up -d
```

or just the proton-bridge:

```bash
docker compose -f docker-compose.dev.yml up protonmail-bridge
```

## Real-World Examples

### Proton Mail Notifications

Receive Signal notifications when new emails arrive in your Proton Mail inbox.

Prism monitors a Proton Mail account via the local bridge and forwards email alerts through Signal. This relies on the same technology that a third-party email client like Thunderbird would be using to integrate with Proton Mail.

### Home Assistant Alerts

Add a rest notification configuration (eg. add to configuration.yaml) to Home Assistant like:

```yaml
notify:
  - platform: rest
    name: Prism
    resource: "http://<Your Prism server network IP>/Home Assistant"
    method: POST
    headers:
      Authorization: !secret prism_api_key
```

Since Home Assistant and Prism are both on your local network, HTTP is allowed automatically - no additional configuration needed.

Add your API_KEY to your secrets.yaml:

```bash
prism_api_key: "Bearer YOUR_API_KEY_HERE"
```

Reboot your Home Assistant system and you'll then be able to send Signal notifications to yourself by using this notify prism action.

## API Reference

### Send Notification

#### POST /{topic}

Send a notification to a registered app/topic. Compatible with ntfy format.

```bash
curl -X POST http://localhost:8080/my-app \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"title": "Alert", "message": "Something happened"}'
```

Or ntfy-style:

```bash
curl -X POST http://localhost:8080/my-app \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d "Simple message text"
```

### WebPush/Webhook Management

#### POST /webpush/app

Register or update a WebPush subscription or plain webhook.

Encrypted WebPush (all crypto fields required):

```bash
curl -X POST http://localhost:8080/webpush/app \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "appName": "my-app",
    "pushEndpoint": "https://updates.push.services.mozilla.org/...",
    "p256dh": "base64-encoded-key",
    "auth": "base64-encoded-auth",
    "vapidPrivateKey": "base64-encoded-vapid-key"
  }'
```

Plain HTTP webhook (no encryption):

```bash
curl -X POST http://localhost:8080/webpush/app \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "appName": "my-app",
    "pushEndpoint": "https://your-server.com/webhook"
  }'
```

#### DELETE /webpush/app/{appName}

Unregister a WebPush subscription (clears WebPush settings, reverts to Signal).

```bash
curl -X DELETE http://localhost:8080/webpush/app/my-app \
  -H "Authorization: Bearer YOUR_API_KEY"
```

## Monitoring

The health of the system can be viewed in the same admin UI used for linking Signal. Prism uses [basic access authentication](https://en.wikipedia.org/wiki/Basic_access_authentication) - provide your `API_KEY` as the password (username can be anything).

For API-based monitoring, call `/api/health` which returns JSON:

```json
{"uptime":"3s","signal":{"daemon":"running","linked":true},"protonMail":"connected"}
```

## Architecture

Prism accepts notifications via HTTP POST requests and routes them based on your configured delivery method:

- **Signal groups**: Uses [signal-cli](https://github.com/AsamK/signal-cli) to create a Signal group for each app and send notifications as messages
- **WebPush**: Supports both encrypted WebPush (with VAPID signing and payload encryption) and plain HTTP webhooks
  - **Encrypted WebPush**: Full WebPush protocol with end-to-end encryption - requires `appName`, `pushEndpoint`, `p256dh`, `auth`, and `vapidPrivateKey`
  - **Plain webhooks**: Simple JSON POST to any HTTP endpoint - only requires `appName` and `pushEndpoint`

Each app can be independently configured to use either delivery method through the admin UI.

For the optional Proton Mail integration, Prism requires a server that runs Proton's official [proton-bridge](https://github.com/ProtonMail/proton-bridge). Prism's docker compose process will run an image from [protonmail-bridge-docker](https://github.com/shenxn/protonmail-bridge-docker). Once authenticated, the communication between Prism and proton-bridge will be over IMAP.
