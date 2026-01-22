import { Hono } from 'hono';
import { basicAuth } from 'hono/basic-auth';
import { DEVICE_NAME, PROTON_IMAP_PASSWORD, PROTON_IMAP_USERNAME } from '@/constants/config';
import { isImapConnected } from '@/modules/proton-mail';
import {
  checkSignalCli,
  finishLink,
  generateLinkQR,
  getAccount,
  hasValidAccount,
} from '@/modules/signal';
import { getAllMappings, remove } from '@/modules/store';
import { verifyApiKey } from '@/utils/auth';
import { formatPhoneNumber, formatUptime } from '@/utils/format';

let cachedQR: string | null = null;
let qrCacheTime = 0;
let generatingPromise: Promise<string> | null = null;
const QR_CACHE_TTL = 10 * 60 * 1000;

export const admin = new Hono();

admin.use(
  '*',
  basicAuth({
    verifyUser: (_, password, c) => verifyApiKey(password, c),
    realm: 'SUP Admin - Username: any, Password: API_KEY',
  }),
);

admin.get('/api/health', async (c) => {
  const signalOk = await checkSignalCli();
  const linked = signalOk && (await hasValidAccount());
  const hasProtonConfig = PROTON_IMAP_USERNAME && PROTON_IMAP_PASSWORD;

  const result: Record<string, unknown> = {
    uptime: formatUptime(process.uptime()),
    signal: {
      daemon: signalOk ? 'running' : 'stopped',
      linked,
    },
  };

  if (hasProtonConfig) {
    result.protonMail = isImapConnected() ? 'connected' : 'disconnected';
  }

  return c.json(result);
});

admin.get('/fragment/health', async (c) => {
  const { html } = await handleHealthFragment();
  return c.html(html);
});

admin.get('/fragment/signal-info', async (c) => c.html(await handleSignalInfoFragment()));

admin.get('/fragment/endpoints', async (c) => c.html(await handleEndpointsFragment()));

admin.get('/fragment/link-qr', async (c) => c.html(await handleQRSection()));

admin.delete('/action/delete-endpoint/:endpoint', async (c) => {
  const endpoint = decodeURIComponent(c.req.param('endpoint'));

  if (!endpoint) {
    return c.text('Invalid endpoint', 400);
  }

  remove(endpoint);
  return c.html(await handleEndpointsFragment());
});

const handleHealthFragment = async () => {
  const signalOk = await checkSignalCli();
  const linked = signalOk && (await hasValidAccount());
  const imap = isImapConnected();
  const hasProtonConfig = PROTON_IMAP_USERNAME && PROTON_IMAP_PASSWORD;
  const accountNumber = getAccount();

  const html = `
    <div class="status">
      <div class="status-item ${signalOk ? 'status-ok' : 'status-error'}">
        Signal Network: ${signalOk ? 'Connected' : 'Disconnected'}
      </div>
      <div class="status-item ${linked ? 'status-ok' : 'status-error'}">
        Account: ${linked ? 'Linked' : 'Unlinked'}
        ${linked && accountNumber ? `<span class="tooltip">${formatPhoneNumber(accountNumber)}</span>` : ''}
      </div>
      ${
        hasProtonConfig
          ? `<div class="status-item ${imap ? 'status-ok' : 'status-error'}">
        Proton Mail: ${imap ? 'Connected' : 'Disconnected'}
        ${imap ? `<span class="tooltip">${PROTON_IMAP_USERNAME}</span>` : ''}
      </div>`
          : ''
      }
    </div>
    <div id="signal-info" hx-swap-oob="true">
      ${await handleSignalInfoFragment()}
    </div>
  `;

  return { html, linked };
};

const handleSignalInfoFragment = async () => {
  if (await hasValidAccount()) {
    cachedQR = null;
    return `<details class="unlink-details">
         <summary class="unlink-summary">Unlink and remove device</summary>
         <div class="unlink-instructions">
           <ol>
             <li>Open Signal app → <strong>Settings → Linked Devices</strong></li>
             <li>Find <strong>"${DEVICE_NAME}"</strong> and tap it</li>
             <li>Tap <strong>"Unlink Device"</strong></li>
           </ol>
         </div>
       </details>`;
  }

  return handleQRSection();
};

const handleEndpointsFragment = async () => {
  const endpoints = getAllMappings();

  if (endpoints.length === 0) {
    return '<p>No endpoints registered</p>';
  }

  return `
    <ul class="endpoint-list">
      ${endpoints
        .map(
          (e: { appName: string; endpoint: string }) => `
        <li class="endpoint-item">
          <div class="endpoint-name">
            <strong>${e.appName}</strong>
          </div>
          <button 
            class="btn-delete"
            hx-delete="/action/delete-endpoint/${encodeURIComponent(e.endpoint)}"
            hx-target="#endpoints-list"
            hx-swap="innerHTML"
          >Delete</button>
        </li>
      `,
        )
        .join('')}
    </ul>
  `;
};

const handleQRSection = async () => {
  if (await hasValidAccount()) {
    return '<p>Account already linked</p>';
  }

  const now = Date.now();

  if ((!cachedQR || now - qrCacheTime > QR_CACHE_TTL) && !generatingPromise) {
    generatingPromise = (async () => {
      try {
        const qr = await generateLinkQR();
        cachedQR = qr;
        qrCacheTime = Date.now();

        finishLink().finally(() => {
          generatingPromise = null;
          cachedQR = null;
          qrCacheTime = 0;
        });

        return qr;
      } catch (error) {
        generatingPromise = null;
        throw error;
      }
    })();
  }

  if (generatingPromise && !cachedQR) {
    try {
      await generatingPromise;
    } catch {
      return '<p>Signal daemon is starting up, please refresh in a few seconds...</p>';
    }
  }

  return `
    <p>Scan this QR code with your Signal app:</p>
    <p class="qr-instructions"><strong>Settings → Linked Devices → Link New Device</strong></p>
    <div class="qr-container">
      <img src="${cachedQR}" class="qr-image" alt="QR Code" />
    </div>
  `;
};
