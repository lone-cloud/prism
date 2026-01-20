import { DEVICE_NAME } from '@/constants/config';
import { SIGNAL_CLI_DATA } from '@/constants/paths';
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
import { getAllMappings } from '@/modules/store';

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
        IMAP: ${imap ? 'Connected' : 'Disconnected'}
      </div>
    </div>
  `;

  return new Response(html, {
    headers: { 'content-type': 'text/html' },
  });
};

export const handleSignalInfoFragment = async () => {
  const linked = await hasValidAccount();

  const html = linked
    ? `<details style="margin-top: 15px;">
         <summary style="cursor: pointer; font-weight: bold;">Unlink and remove device</summary>
         <div style="margin-top: 15px;">
           <ol style="margin-left: 20px;">
             <li>Open Signal app → <strong>Settings → Linked Devices</strong></li>
             <li>Find <strong>"${DEVICE_NAME}"</strong> and tap it</li>
             <li>Tap <strong>"Unlink Device"</strong></li>
           </ol>
         </div>
       </details>`
    : `<button onclick="document.getElementById('qr-section').style.display='block'; 
                        this.parentElement.style.display='none';
                        htmx.ajax('GET', '/link/qr-section', {target: '#qr-section', swap: 'innerHTML'})"
               class="link-button" style="border:none;cursor:pointer;">
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
      ${endpoints.map((e) => `<li><strong>${e.appName}</strong><br><code>${e.endpoint}</code> → ${e.groupId}</li>`).join('')}
    </ul>
  `;

  return new Response(html, {
    headers: { 'content-type': 'text/html' },
  });
};

export const handleQRSection = async () => {
  const html = `
    <p>Scan this QR code with your Signal app:</p>
    <p style="font-size: 1.05em;"><strong>Settings → Linked Devices → Link New Device</strong></p>
    <div hx-get="/link/qr-image" hx-trigger="load, every 30s" style="margin-top:15px;">
      Generating QR code...
    </div>
    <div hx-get="/link/status-check" hx-trigger="every 2s" hx-swap="none"></div>
    <button onclick="document.getElementById('qr-section').style.display='none'; 
                     document.getElementById('signal-info').style.display='block'"
            style="margin-top:15px;padding:8px 16px;background:#6c757d;color:white;border:none;border-radius:4px;cursor:pointer;">
      Cancel
    </button>
  `;

  return new Response(html, {
    headers: { 'content-type': 'text/html' },
  });
};

export const handleQRImage = async () => {
  const linked = await hasValidAccount();
  if (!linked && (await Bun.file(SIGNAL_CLI_DATA).exists())) {
    await unlinkDevice();
    await restartDaemon();
  }

  const qrDataUrl = await generateLinkQR();
  const html = `<img src="${qrDataUrl}" style="max-width: 300px;" alt="QR Code" />`;

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
