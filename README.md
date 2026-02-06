<div align="center">

<img src="assets/prism.webp" alt="Prism Icon" width="120" height="120" />

# Prism

**Self-hosted notification gateway using WebPush and optional Signal for transport**

[Setup](#setup) • [Real-World Examples](#real-world-examples) • [Architecture](#architecture)

</div>

<!-- markdownlint-enable MD033 -->

Prism is a self-hosted notification gateway that receives HTTP requests and routes them through WebPush apps or optionally through Signal groups. Route notifications through Signal to avoid exposing unique network fingerprints, or forward them to your own WebPush apps for custom handling.

## Setup

```bash
# Download docker-compose.yml
curl -L -O https://raw.githubusercontent.com/lone-cloud/prism/master/docker-compose.yml

# Download .env.example
curl -L -O https://raw.githubusercontent.com/lone-cloud/prism/master/.env.example

# Configure your API key (eg. admin password)
cp .env.example .env
nano .env  # Set API_KEY=your-secret-key-here

# Start Prism
docker compose up -d
```

Prism is now running at <http://localhost:8080>. By default, all notifications use WebPush (encrypted push notifications or plain HTTP webhooks). Enable optional integrations below for Signal or Telegram delivery, or Proton Mail monitoring.

## Integrations

All integrations are optional. Enable only what you need.

### Signal

Send notifications through Signal groups instead of WebPush.

**1. Enable Signal in `.env`:**

```bash
FEATURE_ENABLE_SIGNAL=true
```

**2. Start Prism with Signal:**

```bash
docker compose --profile signal up -d
```

**3. Link your Signal account:**

Visit <http://localhost:8080>, authenticate with your API_KEY, and scan the QR code from your Signal app:

**Settings → Linked Devices → Link New Device**

![QR code linking screen](assets/screenshots/2.webp)

Once linked, new apps will default to Signal delivery.

### Telegram

Send notifications through Telegram instead of WebPush.

**1. Create a Telegram bot:**

- Message [@BotFather](https://t.me/BotFather) on Telegram
- Send `/newbot` and follow the prompts
- Copy the bot token (looks like `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`)

**2. Enable Telegram in `.env`:**

```bash
FEATURE_ENABLE_TELEGRAM=true
TELEGRAM_BOT_TOKEN=your-bot-token-here
```

**3. Get your chat ID:**

- Message [@userinfobot](https://t.me/userinfobot) on Telegram
- Copy your Chat ID from the bot's response

**4. Add chat ID to `.env`:**

```bash
TELEGRAM_CHAT_ID=123456789
```

**5. Start Prism:**

```bash
# Telegram runs in the main Prism service (no profile needed)
docker compose up -d
```

All notifications will now be sent to your Telegram chat. Unlike Signal, Telegram doesn't create separate groups per app - all notifications go to the configured chat ID.

### Proton Mail

Receive notifications when new Proton Mail emails arrive.

> **Note:** The default image (`shenxn/protonmail-bridge:build`) compiles from source and supports all architectures. For x86_64 only, you can use `shenxn/protonmail-bridge:latest` (smaller, faster).

**1. Initialize the bridge:**

```bash
docker compose run --rm protonmail-bridge init
```

**2. Login at the `>>>` prompt:**

```bash
>>> login
# Enter your Proton Mail email
# Enter your password
# Enter your 2FA code
```

**3. Get IMAP credentials:**

```bash
>>> info
# Copy the Username and Password
>>> exit
```

**4. Add credentials to `.env`:**

```bash
FEATURE_ENABLE_PROTON=true
PROTON_IMAP_USERNAME=username-from-info-command
PROTON_IMAP_PASSWORD=password-from-info-command
```

**5. Start Prism with Proton Mail:**

```bash
# Start only Prism + Proton Mail
docker compose --profile proton up -d

# Or with Signal + Proton Mail
docker compose --profile signal --profile proton up -d
```

Prism will now forward Proton Mail notifications to your configured channel (Signal, Telegram, or WebPush).

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

- **WebPush** (default): Supports both encrypted WebPush (with VAPID signing and payload encryption) and plain HTTP webhooks
  - **Encrypted WebPush**: Full WebPush protocol with end-to-end encryption - requires `appName`, `pushEndpoint`, `p256dh`, `auth`, and `vapidPrivateKey`
  - **Plain webhooks**: Simple JSON POST to any HTTP endpoint - only requires `appName` and `pushEndpoint`
- **Signal groups** (optional): Uses [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) to create a Signal group for each app and send notifications as messages

Each app can be independently configured to use either delivery method through the admin UI.

For the optional Proton Mail integration, Prism requires a server that runs Proton's official [proton-bridge](https://github.com/ProtonMail/proton-bridge). Prism's docker compose process will run an image from [protonmail-bridge-docker](https://github.com/shenxn/protonmail-bridge-docker). Once authenticated, the communication between Prism and proton-bridge will be over IMAP.
