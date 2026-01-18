import chalk from 'chalk';
import { API_KEY, BRIDGE_IMAP_PASSWORD, BRIDGE_IMAP_USERNAME, PORT } from './constants/config';
import { CONTENT_TYPE, ROUTES, TEMPLATES } from './constants/server';
import { checkSignalCli, hasValidAccount, initSignal, startDaemon } from './modules/signal';
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
import { checkAuth } from './utils/auth';

let daemon: ReturnType<typeof Bun.spawn> | null = null;

daemon = await startDaemon();

const isLinked = await checkSignalCli();
const hasAccount = isLinked && (await initSignal({}));

if (hasAccount) {
  console.log(chalk.green('✓ Signal account linked'));
} else {
  console.log(chalk.yellow('⚠ No Signal account linked'));
  console.log(chalk.dim(`  Visit http://localhost:${PORT}/link to link your device`));
}

if (!API_KEY) {
  console.warn(chalk.yellow('⚠️  Server running without API_KEY'));
  console.warn(chalk.dim('   Set API_KEY env var for production deployments.'));
}

if (BRIDGE_IMAP_USERNAME && BRIDGE_IMAP_PASSWORD) {
  const { startProtonMonitor } = await import('./modules/protonmail');
  await startProtonMonitor();
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
      GET: handleLinkQR,
    },

    [ROUTES.LINK_STATUS]: {
      GET: handleLinkStatus,
    },

    [ROUTES.LINK_UNLINK]: {
      POST: async (req) => {
        const response = await handleUnlink(req, daemon);
        if (response.status === 303) {
          daemon = await startDaemon();
        }
        return response;
      },
    },

    [ROUTES.UP]: {
      GET: handleDiscovery,
    },

    [ROUTES.ENDPOINTS]: {
      GET: (req) => {
        const auth = checkAuth(req);
        if (auth) return auth;
        return handleEndpoints();
      },
    },

    [ROUTES.TOPICS]: {
      GET: (req) => {
        const auth = checkAuth(req);
        if (auth) return auth;
        return handleTopics();
      },
    },

    [ROUTES.MATRIX_NOTIFY]: {
      POST: handleMatrixNotify,
    },

    [ROUTES.UP_INSTANCE]: {
      POST: (req) => {
        const auth = checkAuth(req);
        if (auth) return auth;
        return handleRegister(req, new URL(req.url));
      },
      DELETE: (req) => {
        const auth = checkAuth(req);
        if (auth) return auth;
        return handleUnregister(new URL(req.url));
      },
    },

    [ROUTES.NOTIFY_TOPIC]: {
      POST: (req) => {
        const auth = checkAuth(req);
        if (auth) return auth;
        return handleNotify(req, new URL(req.url));
      },
    },
  },

  async fetch(_req) {
    if (!(await hasValidAccount())) {
      const html = await Bun.file(TEMPLATES.SETUP).text();
      return new Response(html, { headers: { 'content-type': CONTENT_TYPE.HTML } });
    }

    return new Response(null, { status: 404 });
  },
});

console.log(chalk.cyan.bold(`\n🚀 SUP running on http://localhost:${server.port}`));
