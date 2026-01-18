import chalk from 'chalk';
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

let daemon: ReturnType<typeof Bun.spawn> | null = null;

try {
  daemon = await startDaemon();
  const isLinked = await checkSignalCli();
  const hasAccount = isLinked && (await initSignal({}));

  if (hasAccount) {
    console.log(chalk.green('✓ Signal account linked'));
  } else {
    console.log(chalk.yellow('⚠ No Signal account linked'));
    console.log(chalk.dim(`  Visit http://localhost:${PORT}/link to link your device`));
  }
} catch (error) {
  console.error(chalk.red('✗ Failed to start signal-cli daemon'));
  console.error(chalk.dim(`  ${error instanceof Error ? error.message : String(error)}`));
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

console.log(chalk.cyan.bold(`\n🚀 SUP running on http://localhost:${server.port}`));
