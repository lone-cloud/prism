import { API_KEY, BRIDGE_IMAP_PASSWORD, BRIDGE_IMAP_USERNAME, PORT } from '@/constants/config';
import { PUBLIC_DIR } from '@/constants/paths';
import { cleanupDaemon, initSignal } from '@/modules/signal';
import { adminRoutes } from '@/routes/admin';
import { ntfyRoutes } from '@/routes/ntfy';
import { unifiedPushRoutes } from '@/routes/unifiedpush';
import { maybeCompress } from '@/utils/compress';
import { getLanIP } from '@/utils/ip';
import { logError, logInfo, logVerbose, logWarn } from '@/utils/log';

try {
  await initSignal();
} catch (error) {
  logError(`${error instanceof Error ? error.message : String(error)}`);
}

if (!API_KEY) {
  logWarn('Server running without API_KEY');
}

if (BRIDGE_IMAP_USERNAME && BRIDGE_IMAP_PASSWORD) {
  try {
    const { startProtonMonitor } = await import('./modules/protonmail');
    await startProtonMonitor();
  } catch (err) {
    logError('Failed to start Proton Mail monitor:', err);
    logWarn('Continuing without Proton Mail integration');
  }
}

const server = Bun.serve({
  port: PORT,
  idleTimeout: 60,
  maxRequestBodySize: 1024 * 1024, // 1MB limit

  error(error) {
    logError('Unhandled server error:', error);
    return new Response('Internal Server Error', { status: 500 });
  },

  routes: {
    '/favicon.webp': {
      GET: () => new Response(Bun.file(`${PUBLIC_DIR}/favicon.webp`)),
    },
    '/htmx.js': {
      GET: async (req) =>
        maybeCompress(req, await Bun.file(`${PUBLIC_DIR}/htmx.min.js`).text(), 'text/javascript'),
    },

    ...adminRoutes,

    ...unifiedPushRoutes,

    ...ntfyRoutes,
  },
});

logInfo(`\nSUP running on:`);
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
