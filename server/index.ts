import chalk from 'chalk';
import { CONTENT_TYPE, ROUTES, TEMPLATES } from './constants/server';
import {
  checkSignalCli,
  createGroup,
  finishLink,
  generateLinkQR,
  hasValidAccount,
  initSignal,
  sendGroupMessage,
  startDaemon,
  unlinkDevice,
} from './signal';
import { getAllMappings, getGroupId, register, remove } from './store';
import { formatAsSignalMessage, parseUnifiedPushRequest } from './unifiedpush';

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

const getBaseUrl = (req: Request) => {
  const proto = req.headers.get('x-forwarded-proto') || 'http';
  const host = req.headers.get('host') || `localhost:${PORT}`;
  return `${proto}://${host}`;
};

const server = Bun.serve({
  port: PORT,
  idleTimeout: 60,

  async fetch(req) {
    const url = new URL(req.url);

    if (url.pathname === ROUTES.FAVICON) {
      const file = Bun.file('server/assets/favicon.png');
      return new Response(file, {
        headers: { 'content-type': 'image/png' },
      });
    }

    if (url.pathname === ROUTES.HEALTH) {
      const signalOk = await checkSignalCli();
      const linked = signalOk && (await hasValidAccount());
      return Response.json({
        status: 'ok',
        signal: signalOk ? 'connected' : 'disconnected',
        linked,
        mode: 'daemon',
      });
    }

    if (url.pathname === ROUTES.LINK) {
      const linked = await hasValidAccount();
      if (linked) {
        let html = await Bun.file(TEMPLATES.LINKED).text();
        const passwordField = API_KEY
          ? '<input type="password" name="password" placeholder="Enter API_KEY" required style="padding: 8px; margin-right: 10px; border: 1px solid #ccc; border-radius: 4px;" />'
          : '';
        html = html.replace('{{PASSWORD_FIELD}}', passwordField);

        return new Response(html, {
          headers: { 'content-type': CONTENT_TYPE.HTML },
        });
      }

      const html = await Bun.file(TEMPLATES.LINK).text();
      return new Response(html, {
        headers: { 'content-type': CONTENT_TYPE.HTML },
      });
    }

    if (url.pathname === ROUTES.LINK_QR) {
      const qrDataUrl = await generateLinkQR();
      return new Response(qrDataUrl, {
        headers: { 'content-type': CONTENT_TYPE.TEXT },
      });
    }

    if (url.pathname === ROUTES.LINK_STATUS) {
      let linked = await hasValidAccount();

      if (!linked) {
        try {
          await finishLink();
          await initSignal({});
          linked = true;
        } catch {
          // Not ready yet or failed
        }
      }

      return Response.json({ linked });
    }

    if (url.pathname === ROUTES.LINK_UNLINK && req.method === 'POST') {
      if (API_KEY) {
        const formData = await req.formData();
        const password = formData.get('password');

        if (password !== API_KEY) {
          return new Response('Invalid password', { status: 403 });
        }
      }

      await unlinkDevice();

      if (daemon) {
        daemon.kill();
      }

      await new Promise((resolve) => setTimeout(resolve, 500));
      daemon = await startDaemon();

      return new Response('', {
        status: 303,
        headers: { Location: ROUTES.LINK },
      });
    }

    if (!(await hasValidAccount())) {
      const html = await Bun.file(TEMPLATES.SETUP).text();
      return new Response(html, {
        headers: { 'content-type': CONTENT_TYPE.HTML },
      });
    }

    if (url.pathname === ROUTES.MATRIX_NOTIFY && req.method === 'POST') {
      const message = await parseUnifiedPushRequest(req);
      const groupId = getGroupId(message.endpoint);

      if (!groupId) {
        return new Response('Endpoint not registered', { status: 404 });
      }

      const signalMessage = formatAsSignalMessage(message);
      await sendGroupMessage(groupId, signalMessage);

      return Response.json({ success: true });
    }

    if (url.pathname.startsWith(ROUTES.UP_PREFIX) && req.method === 'POST') {
      if (API_KEY && req.headers.get('authorization') !== `Bearer ${API_KEY}`) {
        return new Response('Unauthorized', { status: 401 });
      }
      const endpointId = url.pathname.split('/')[2] ?? '';
      const { appName } = (await req.json()) as {
        appName: string;
        token?: string;
      };

      const groupId: string = getGroupId(endpointId) ?? (await createGroup(`SUP - ${appName}`));

      if (!getGroupId(endpointId)) {
        register(endpointId, groupId, appName);
      }

      const baseUrl = getBaseUrl(req);
      const endpoint = `${baseUrl}${ROUTES.MATRIX_NOTIFY}/${endpointId}`;

      return Response.json({ endpoint, gateway: 'matrix' });
    }

    if (url.pathname.startsWith(ROUTES.UP_PREFIX) && req.method === 'DELETE') {
      const endpointId = url.pathname.split('/')[2] ?? '';
      remove(endpointId);
      return new Response('', { status: 204 });
    }

    if (url.pathname === ROUTES.UP && req.method === 'GET') {
      return Response.json({
        unifiedpush: { version: 1 },
        gateway: 'matrix',
      });
    }

    if (url.pathname === ROUTES.ENDPOINTS && req.method === 'GET') {
      return Response.json(getAllMappings());
    }

    return new Response('Not Found', { status: 404 });
  },
});

console.log(chalk.cyan.bold(`\n🚀 SUP running on http://localhost:${server.port}`));
