import chalk from 'chalk';
import { CONTENT_TYPE, ROUTES, TEMPLATES } from './constants/server';
import { handleHealth } from './routes/health';
import { handleLink, handleLinkQR, handleLinkStatus, handleUnlink } from './routes/link';
import { handleGetNotifications, handleNotify, handleTopics } from './routes/notify';
import {
  handleDiscovery,
  handleEndpoints,
  handleMatrixNotify,
  handleRegister,
  handleUnregister,
} from './routes/unifiedpush';
import { checkSignalCli, hasValidAccount, initSignal, startDaemon } from './signal';

const PORT = Bun.env.PORT || 8080;
const API_KEY = Bun.env.API_KEY;

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
  console.warn(chalk.yellow('⚠️  Server running without API_KEY - anyone can register endpoints!'));
  console.warn(chalk.dim('   Set API_KEY env var for production deployments.'));
}

const requireHttps = (req: Request) => {
  const proto = req.headers.get('x-forwarded-proto') || 'http';
  const host = req.headers.get('host') || '';
  const isLocalhost = host.startsWith('localhost') || host.startsWith('127.0.0.1');

  if (API_KEY && proto !== 'https' && !isLocalhost) {
    return new Response('HTTPS required when API_KEY is configured', { status: 403 });
  }

  return null;
};

const server = Bun.serve({
  port: PORT,
  idleTimeout: 60,

  async fetch(req) {
    const url = new URL(req.url);

    if (url.pathname === ROUTES.FAVICON) {
      const file = Bun.file('server/assets/favicon.png');
      return new Response(file, { headers: { 'content-type': 'image/png' } });
    }

    if (url.pathname === ROUTES.HEALTH) return handleHealth();
    if (url.pathname === ROUTES.LINK) return handleLink();
    if (url.pathname === ROUTES.LINK_QR) return handleLinkQR();
    if (url.pathname === ROUTES.LINK_STATUS) return handleLinkStatus();

    if (!(await hasValidAccount())) {
      const html = await Bun.file(TEMPLATES.SETUP).text();
      return new Response(html, { headers: { 'content-type': CONTENT_TYPE.HTML } });
    }

    if (url.pathname === ROUTES.MATRIX_NOTIFY && req.method === 'POST') {
      return handleMatrixNotify(req);
    }

    const httpsCheck = requireHttps(req);
    if (httpsCheck) return httpsCheck;

    if (url.pathname === ROUTES.LINK_UNLINK && req.method === 'POST') {
      const response = await handleUnlink(req, daemon);

      if (response.status === 303) {
        daemon = await startDaemon();
      }

      return response;
    }

    if (url.pathname.startsWith(ROUTES.UP_PREFIX)) {
      if (req.method === 'POST') return handleRegister(req, url);
      if (req.method === 'DELETE') return handleUnregister(url);
    }

    if (url.pathname === ROUTES.UP && req.method === 'GET') return handleDiscovery();
    if (url.pathname === ROUTES.ENDPOINTS && req.method === 'GET') return handleEndpoints();
    if (url.pathname === ROUTES.TOPICS && req.method === 'GET') return handleTopics();
    if (url.pathname === ROUTES.NOTIFICATIONS && req.method === 'GET') {
      return handleGetNotifications(req, url);
    }

    if (url.pathname.startsWith(ROUTES.NOTIFY_PREFIX) && req.method === 'POST') {
      return handleNotify(req, url);
    }

    return new Response(null, { status: 404 });
  },
});

console.log(chalk.cyan.bold(`\n🚀 SUP running on http://localhost:${server.port}`));
