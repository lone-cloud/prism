import { DEVICE_NAME } from '@/constants/config';
import { PUBLIC_DIR, SIGNAL_CLI_DATA } from '@/constants/paths';
import { isImapConnected } from '@/modules/protonmail';
import {
  checkSignalCli,
  finishLink,
  generateLinkQR,
  hasLinkUri,
  hasValidAccount,
  initSignal,
  restartDaemon,
  unlinkDevice,
} from '@/modules/signal';
import { getAllMappings, remove } from '@/modules/store';
import { withAuth } from '@/utils/auth';
import { maybeCompress } from '@/utils/compress';

export const handleHealthFragment = async () => {
  const signalOk = await checkSignalCli();
  const linked = signalOk && (await hasValidAccount());
  const imap = isImapConnected();

  const html = `
    <div class="status">
      <div class="status-item ${signalOk ? 'status-ok' : 'status-error'}">
        Signal: ${signalOk ? 'Connected' : 'Disconnected'}
      </div>
      <div class="status-item ${linked ? 'status-ok' : 'status-error'}">
        Account: ${linked ? 'Linked' : 'Unlinked'}
      </div>
      <div class="status-item ${imap ? 'status-ok' : 'status-error'}">
        Proton Mail: ${imap ? 'Connected' : 'Disconnected'}
      </div>
    </div>
  `;

  return new Response(html, {
    headers: { 'content-type': 'text/html' },
  });
};

export const handleSignalInfoFragment = async () => {
  const html = (await hasValidAccount())
    ? `<details class="unlink-details">
         <summary class="unlink-summary">Unlink and remove device</summary>
         <div class="unlink-instructions">
           <ol>
             <li>Open Signal app → <strong>Settings → Linked Devices</strong></li>
             <li>Find <strong>"${DEVICE_NAME}"</strong> and tap it</li>
             <li>Tap <strong>"Unlink Device"</strong></li>
           </ol>
         </div>
       </details>`
    : `<button onclick="document.getElementById('qr-section').style.display='block'; 
                        this.parentElement.style.display='none';
                        htmx.ajax('GET', '/link/qr-section', {target: '#qr-section', swap: 'innerHTML'})"
               class="link-button">
         Link Signal Device
       </button>`;

  return new Response(html, {
    headers: { 'content-type': 'text/html' },
  });
};

export const handleEndpointsFragment = async () => {
  const endpoints = getAllMappings();

  if (endpoints.length === 0) {
    return new Response('<p>No endpoints registered</p>', {
      headers: { 'content-type': 'text/html' },
    });
  }

  const html = `
    <ul class="endpoint-list">
      ${endpoints
        .map(
          (e) => `
        <li class="endpoint-item">
          <div class="endpoint-name">
            <strong>${e.appName}</strong>
          </div>
          <button 
            class="btn-delete"
            hx-delete="/endpoint/delete/${encodeURIComponent(e.endpoint)}"
            hx-target="#endpoints-list"
            hx-swap="innerHTML"
          >Delete</button>
        </li>
      `,
        )
        .join('')}
    </ul>
  `;

  return new Response(html, {
    headers: { 'content-type': 'text/html' },
  });
};

export const handleQRSection = async () => {
  const html = `
    <p>Scan this QR code with your Signal app:</p>
    <p class="qr-instructions"><strong>Settings → Linked Devices → Link New Device</strong></p>
    <div hx-get="/link/qr-image" hx-trigger="load, every 30s" class="qr-container">
      Generating QR code...
    </div>
    <div hx-get="/link/status-check" hx-trigger="every 2s" hx-swap="none"></div>
    <button onclick="document.getElementById('qr-section').style.display='none'; 
                     document.getElementById('signal-info').style.display='block'"
            class="btn-cancel">
      Cancel
    </button>
  `;

  return new Response(html, {
    headers: { 'content-type': 'text/html' },
  });
};

export const handleQRImage = async () => {
  if (!(await hasValidAccount()) && (await Bun.file(SIGNAL_CLI_DATA).exists())) {
    await unlinkDevice();
    await restartDaemon();
  }

  const qrDataUrl = await generateLinkQR();
  const html = `<img src="${qrDataUrl}" class="qr-image" alt="QR Code" />`;

  return new Response(html, {
    headers: { 'content-type': 'text/html' },
  });
};

export const handleLinkStatusCheck = async () => {
  let linked = await hasValidAccount();

  if (!linked && hasLinkUri()) {
    try {
      await finishLink();
      const result = await initSignal();
      linked = result.linked;
    } catch (error) {
      console.error('Failed to finish link:', error);
    }
  }

  if (linked) {
    return new Response('', {
      headers: {
        'content-type': 'text/html',
        'HX-Refresh': 'true',
      },
    });
  }

  return new Response('', {
    headers: { 'content-type': 'text/html' },
  });
};

export const handleDeleteEndpoint = async (req: Request) => {
  const url = new URL(req.url);
  const endpoint = decodeURIComponent(url.pathname.split('/').pop() || '');

  if (!endpoint) {
    return new Response('Invalid endpoint', { status: 400 });
  }

  remove(endpoint);

  return handleEndpointsFragment();
};

export const adminRoutes = {
  '/': {
    GET: withAuth(async (req: Request) =>
      maybeCompress(req, await Bun.file(`${PUBLIC_DIR}/admin.html`).text()),
    ),
  },

  '/admin.css': {
    GET: withAuth(async (req: Request) =>
      maybeCompress(req, await Bun.file(`${PUBLIC_DIR}/admin.css`).text(), 'text/css'),
    ),
  },

  '/health/fragment': {
    GET: withAuth(handleHealthFragment),
  },

  '/signal-info/fragment': {
    GET: withAuth(handleSignalInfoFragment),
  },

  '/endpoints/fragment': {
    GET: withAuth(handleEndpointsFragment),
  },

  '/link/qr-section': {
    GET: withAuth(handleQRSection),
  },

  '/link/qr-image': {
    GET: withAuth(handleQRImage),
  },

  '/link/status-check': {
    GET: withAuth(handleLinkStatusCheck),
  },

  '/endpoint/delete/:endpoint': {
    DELETE: withAuth(handleDeleteEndpoint),
  },
};
