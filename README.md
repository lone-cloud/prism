<div align="center">

<img src="assets/prism.webp" alt="Prism Icon" width="120" height="120" />

# Prism

**Self-hosted notification gateway**

[Setup](#setup) • [Real-World Examples](#real-world-examples)

</div>

<!-- markdownlint-enable MD033 -->

Prism is a self-hosted notification gateway. Prism can receive messages and route them to Signal, Telegram or WebPush URLs. Messages can be sent via Webhooks or from an optional Proton Mail integration.

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

Prism is now running at <http://localhost:8080>. Enable optional integrations below for custom functionality.

## Integrations

All integrations are optional. Enable only what you need.

### Signal

Send notifications through Signal groups.
Each registered app will route messages to private Signal groups matching the app name.

**1. Enable Signal in `.env`:**

```bash
FEATURE_ENABLE_SIGNAL=true
```

**2. Start Prism with the Signal service:**

```bash
docker compose --profile signal up -d
```

**3. Link your Signal account:**

Visit <http://localhost:8080>, authenticate with your API_KEY, and scan the QR code from your Signal app:

**Settings → Linked Devices → Link New Device**

Once linked, new apps will default to Signal delivery.

### Telegram

Send notifications through Telegram instead.
Unlike the Signal integration, Telegram relies on creating a Telegram bot as it will serve as the notifications messenger.

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
Unlike other integrations, this one will generate new messages to be delivered by one of the configured transports.
Note that using this integration requires a paid Proton Mail account to be able to use the Proton Mail Bridge that this integration relies on.
Also note that the Proton Mail Bridge is RAM hungry.

> **Note:** The default image (`shenxn/protonmail-bridge:build`) used by Prism compiles from source and supports all architectures. For x86_64 only, you can use `shenxn/protonmail-bridge:latest` (smaller, faster).

**1. Initialize the Proton Mail Bridge:**

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
docker compose --profile proton up -d
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

#### POST /{appName}

Send a notification. Compatible with ntfy format.

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

### Health Endpoints

#### GET /health

Public health check endpoint (no authentication required). Returns `200 OK` when the service is running. Used for Docker health checks and load balancer health probes.

```bash
curl http://localhost:8080/health
```

#### GET /api/health

Detailed health endpoint (requires authentication). Returns JSON with uptime and integration status:

```bash
curl http://localhost:8080/api/health \
  -H "Authorization: Bearer YOUR_API_KEY"
```

```json
{"version":"1.0.0","uptime":"3s","signal":{"linked":true},"proton":{"linked":true},"telegram":{"linked":true}}
```
