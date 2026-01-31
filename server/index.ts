import { Hono } from 'hono';
import { getConnInfo, serveStatic } from 'hono/bun';
import { compress } from 'hono/compress';
import { secureHeaders } from 'hono/secure-headers';
import { timeout } from 'hono/timeout';
import { rateLimiter } from 'hono-rate-limiter';
import {
  ALLOW_INSECURE_HTTP,
  API_KEY,
  PORT,
  PROTON_IMAP_PASSWORD,
  PROTON_IMAP_USERNAME,
  RATE_LIMIT,
} from '@/constants/config';
import { PUBLIC_DIR } from '@/constants/paths';
import { cleanupDaemon, initSignal } from '@/modules/signal';
import { admin } from '@/routes/admin';
import { ntfy } from '@/routes/ntfy';
import { protonMail } from '@/routes/proton-mail';
import { webhook } from '@/routes/webhook';
import { getLanIP, isLocalIP } from '@/utils/auth';
import { formatToCspString } from '@/utils/format';
import { logError, logInfo, logVerbose, logWarn } from '@/utils/log';

const cspConfig = {
  defaultSrc: ["'self'"],
  scriptSrc: ["'self'"],
  styleSrc: ["'self'", "'unsafe-inline'"],
  imgSrc: ["'self'", 'data:'],
  formAction: ["'self'"],
  frameAncestors: ["'none'"],
  objectSrc: ["'none'"],
};

initSignal();

if (!API_KEY) {
  logWarn('Server running without API_KEY');
}

if (PROTON_IMAP_USERNAME && PROTON_IMAP_PASSWORD) {
  try {
    const { startProtonMonitor } = await import('./modules/proton-mail');
    await startProtonMonitor();
  } catch (err) {
    logError('Failed to start Proton Mail monitor:', err);
    logWarn('Continuing without Proton Mail integration');
  }
}

const app = new Hono();

app.use('*', (c, next) => {
  const proto = c.req.header('x-forwarded-proto') || 'http';
  const addr = getConnInfo(c).remote.address;

  if (proto === 'https' || isLocalIP(addr)) {
    return secureHeaders({
      contentSecurityPolicy: cspConfig,
    })(c, next);
  }

  c.header('Content-Security-Policy', formatToCspString(cspConfig));

  return next();
});

app.use('*', timeout(5000));

app.use('*', compress());

app.use(
  '*',
  rateLimiter({
    limit: RATE_LIMIT,
    keyGenerator: (c) => getConnInfo(c).remote.address || 'unknown',
    skip: (c) => ALLOW_INSECURE_HTTP || isLocalIP(getConnInfo(c).remote.address),
  }),
);

app.use('*', serveStatic({ root: PUBLIC_DIR }));

app.route('/', ntfy);
app.route('/', admin);
app.route('/api/proton-mail', protonMail);
app.route('/api/webhook', webhook);

app.notFound((c) => c.text('Not Found', 404));

const server = Bun.serve({
  port: PORT,
  fetch: app.fetch,
  maxRequestBodySize: 1024 * 1024,
  idleTimeout: 30,
});

logInfo(`\nPRISM running on:`);
logInfo(`  Local:   http://localhost:${server.port}`);

const lanIP = getLanIP();
if (lanIP) {
  logVerbose(`  Network: http://${lanIP}:${server.port}\n`);
}

process.on('SIGINT', () => {
  cleanupDaemon();
  process.exit(0);
});

process.on('SIGTERM', () => {
  cleanupDaemon();
  process.exit(0);
});
