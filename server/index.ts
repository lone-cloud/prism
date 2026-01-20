import { API_KEY, BRIDGE_IMAP_PASSWORD, BRIDGE_IMAP_USERNAME, PORT } from '@/constants/config';
import { cleanupDaemon, initSignal } from '@/modules/signal';
import { adminRoutes } from '@/routes/admin/index';
import { ntfyRoutes } from '@/routes/ntfy';
import { unifiedPushRoutes } from '@/routes/unifiedpush';
import { logError, logInfo, logWarn } from '@/utils/log';

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
    logError('Failed to start ProtonMail monitor:', err);
    logWarn('Continuing without ProtonMail integration');
  }
}

const server = Bun.serve({
  port: PORT,
  idleTimeout: 60,

  routes: {
    '/favicon.png': {
      GET: () => new Response(Bun.file('public/favicon.png')),
    },
    '/htmx.js': {
      GET: () => new Response(Bun.file('node_modules/htmx.org/dist/htmx.min.js')),
    },

    ...adminRoutes,

    ...unifiedPushRoutes,

    ...ntfyRoutes,
  },
});

logInfo(`\nSUP running on http://localhost:${server.port} 🚀`);

process.on('SIGINT', () => {
  cleanupDaemon();
  process.exit(0);
});

process.on('SIGTERM', () => {
  cleanupDaemon();
  process.exit(0);
});
