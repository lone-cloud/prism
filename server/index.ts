import { API_KEY, BRIDGE_IMAP_PASSWORD, BRIDGE_IMAP_USERNAME, PORT } from './constants/config';
import { ROUTES } from './constants/server';
import { checkSignalCli, initSignal, startDaemon } from './modules/signal';
import { handleHealth } from './routes/health';
import { handleLink, handleLinkQR, handleLinkStatus, handleUnlink } from './routes/link';
import { handleNotify, handleTopics } from './routes/notify';
import {
  handleDiscovery,
  handleEndpoints,
  handleMatrixNotify,
  handleRegister,
  handleUnregister,
} from './routes/unifiedpush';
import { withAuth, withFormAuth } from './utils/auth';
import { logError, logInfo, logSuccess, logWarn } from './utils/log';

let daemon: ReturnType<typeof Bun.spawn> | null = null;

try {
  daemon = await startDaemon();
  const isLinked = await checkSignalCli();
  const hasAccount = isLinked && (await initSignal({}));

  if (hasAccount) {
    logSuccess('Signal account linked');
  } else {
    logWarn('No Signal account linked');
    logInfo(`Visit http://localhost:${PORT}/link to link your device`);
  }
} catch (error) {
  logError(`  ${error instanceof Error ? error.message : String(error)}`);
}

if (!API_KEY) {
  logWarn('Server running without API_KEY');
  console.warn('Set API_KEY env var for production deployments.');
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
    [ROUTES.FAVICON]: Bun.file('assets/favicon.png'),

    [ROUTES.HEALTH]: handleHealth,

    [ROUTES.LINK]: {
      GET: handleLink,
    },

    [ROUTES.LINK_QR]: {
      GET: () =>
        handleLinkQR(async () => {
          daemon?.kill();
          daemon = await startDaemon();
        }),
    },

    [ROUTES.LINK_STATUS]: {
      GET: handleLinkStatus,
    },

    [ROUTES.LINK_UNLINK]: {
      POST: withFormAuth(() =>
        handleUnlink(async () => {
          daemon?.kill();
          daemon = await startDaemon();
        }),
      ),
    },

    [ROUTES.UP]: {
      GET: handleDiscovery,
    },

    [ROUTES.ENDPOINTS]: {
      GET: withAuth(handleEndpoints),
    },

    [ROUTES.TOPICS]: {
      GET: withAuth(handleTopics),
    },

    [ROUTES.MATRIX_NOTIFY]: {
      POST: handleMatrixNotify,
    },

    [ROUTES.UP_INSTANCE]: {
      POST: withAuth((req) => handleRegister(req, new URL(req.url))),
      DELETE: withAuth((req) => handleUnregister(new URL(req.url))),
    },

    [ROUTES.NOTIFY_TOPIC]: {
      POST: withAuth((req) => handleNotify(req, new URL(req.url))),
    },
  },
});

logInfo(`\nSUP running on http://localhost:${server.port} 🚀`);
