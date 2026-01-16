# SUP ProtonMail Bridge

IMAP bridge that monitors a ProtonMail account via Proton Bridge and sends notifications to a SUP server.

## Architecture

```
ProtonMail (E2EE) → Proton Bridge (IMAP) → This Bridge → SUP Server → Signal → Phone
```

## Prerequisites

1. **Proton Bridge** installed and running locally
   - Download from: https://proton.me/mail/bridge
   - Set up your ProtonMail account
   - Note the generated IMAP password (not your ProtonMail password!)

2. **SUP Server** running
   - See main README for SUP server setup

## Configuration

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Configure your settings:

   ```env
   PROTON_EMAIL=your-email@protonmail.com
   PROTON_PASSWORD=bridge-generated-password  # From Proton Bridge settings
   SUP_SERVER_URL=http://localhost:8080
   SUP_API_KEY=your-api-key                   # Optional
   SUP_TOPIC=protonmail
   ```

## Running Locally

```bash
# Install dependencies
bun install

# Run in development mode (auto-reload)
bun dev

# Run in production mode
bun start
```

## Running with Docker

```bash
# Build
docker build -t sup-proton-bridge .

# Run
docker run -d \
  --name sup-proton-bridge \
  --env-file .env \
  --network host \
  sup-proton-bridge
```

## Running with Docker Compose

See the main `docker-compose.yml` in the repo root.

## How It Works

1. Connects to Proton Bridge via IMAP (localhost:1143)
2. Opens INBOX and enters IDLE mode
3. When new mail arrives, fetches sender and subject
4. Sends notification to SUP server at `/notify/protonmail`
5. SUP delivers notification via Signal to your phone

## Security Notes

- Proton Bridge runs locally and handles E2EE decryption
- This bridge only accesses the already-decrypted IMAP interface
- No credentials are sent to SUP server
- Only email metadata (sender, subject) is transmitted
