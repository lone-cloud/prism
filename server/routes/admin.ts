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
import { getAllMappings, removeEndpoint, updateChannel } from '@/modules/store';
import type { NotificationChannel } from '@/types/notifications';
import { verifyApiKey } from '@/utils/auth';
import { formatPhoneNumber, formatUptime } from '@/utils/format';
import { logError } from '@/utils/log';

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

admin.delete('/action/delete-endpoint', async (c) => {
  const formData = await c.req.formData();
  const endpoint = formData.get('endpoint') as string;

  if (!endpoint) {
    return c.text('Invalid endpoint', 400);
  }

  removeEndpoint(endpoint);

  return c.html(await handleEndpointsFragment());
});

admin.post('/action/toggle-channel', async (c) => {
  try {
    const formData = await c.req.formData();
    const endpoint = formData.get('endpoint') as string;
    const channel = formData.get('channel') as NotificationChannel;

    if (!endpoint) {
      return c.json({ error: 'Invalid endpoint' }, 400);
    }

    if (!channel || !['signal', 'unifiedpush'].includes(channel)) {
      return c.json({ error: 'Invalid channel' }, 400);
    }

    updateChannel(endpoint, channel);

    return c.html(await handleEndpointsFragment());
  } catch (err) {
    logError('Failed to toggle channel:', err);

    return c.text('Invalid request', 400);
  }
});

const handleHealthFragment = async () => {
  const signalOk = await checkSignalCli();
  const linked = signalOk && (await hasValidAccount());
  const imap = isImapConnected();
  const hasProtonConfig = PROTON_IMAP_USERNAME && PROTON_IMAP_PASSWORD;
  const accountNumber = getAccount();

  const html = `
    <div class="status">
      <div class="status-item ${signalOk && linked ? 'status-ok' : 'status-error'}">
        Signal: ${signalOk ? 'Connected' : 'Disconnected'} and ${linked ? 'Linked' : 'Unlinked'}
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
          (e) => `
        <li class="endpoint-item">
          <div class="endpoint-info">
            <div class="endpoint-name">
              <strong>${e.appName}</strong>
            </div>
            <div class="endpoint-channel">
              <span class="channel-badge channel-${e.channel}">${e.channel === 'signal' ? 'Signal' : 'UnifiedPush'}</span>
              ${e.upEndpoint ? `<span class="endpoint-detail">${new URL(e.upEndpoint).hostname}</span>` : ''}
            </div>
          </div>
          <div class="endpoint-actions">
            ${
              e.upEndpoint
                ? `
            <form style="display: inline;">
              <input type="hidden" name="endpoint" value="${e.endpoint.replace(/"/g, '&quot;')}" />
              <select 
                class="channel-select"
                name="channel"
                hx-post="/action/toggle-channel"
                hx-target="#endpoints-list"
                hx-swap="innerHTML"
                hx-include="closest form"
              >
                <option value="signal" ${e.channel === 'signal' ? 'selected' : ''}>Signal</option>
                <option value="unifiedpush" ${e.channel === 'unifiedpush' ? 'selected' : ''}>UnifiedPush</option>
              </select>
            </form>
            `
                : ''
            }
            <form style="display: inline;">
              <input type="hidden" name="endpoint" value="${e.endpoint.replace(/"/g, '&quot;')}" />
              <button 
                class="btn-delete"
                hx-delete="/action/delete-endpoint"
                hx-target="#endpoints-list"
                hx-swap="innerHTML"
                hx-include="closest form"
              >Delete</button>
            </form>
          </div>
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
