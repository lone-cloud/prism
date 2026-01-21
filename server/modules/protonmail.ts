import Imap from 'imap';
import {
  BRIDGE_IMAP_PASSWORD,
  BRIDGE_IMAP_USERNAME,
  PROTON_BRIDGE_HOST,
  PROTON_BRIDGE_PORT,
  SUP_TOPIC,
} from '@/constants/config';
import { hasValidAccount, sendGroupMessage } from '@/modules/signal';
import { getOrCreateGroup } from '@/modules/store';
import { logError, logInfo, logSuccess, logVerbose, logWarn } from '@/utils/log';

let imapConnected = false;

export const isImapConnected = () => imapConnected;

export async function startProtonMonitor() {
  if (!BRIDGE_IMAP_USERNAME || !BRIDGE_IMAP_PASSWORD) {
    logError('Missing required env vars: BRIDGE_IMAP_USERNAME and BRIDGE_IMAP_PASSWORD');
    logWarn('Run: docker compose run --rm protonmail-bridge init');
    logWarn('Then use `login` and `info` commands to get IMAP credentials');

    return;
  }

  if (!(await hasValidAccount())) {
    logWarn('Signal account not linked. Proton Mail notifications will be skipped.');
    logWarn('Link your Signal account at /link to enable email notifications.');
  }

  logInfo(`Connecting to Proton Bridge at ${PROTON_BRIDGE_HOST}:${PROTON_BRIDGE_PORT}`);
  logInfo(`Monitoring mailbox: ${BRIDGE_IMAP_USERNAME}`);

  const imap = new Imap({
    user: BRIDGE_IMAP_USERNAME,
    password: BRIDGE_IMAP_PASSWORD,
    host: PROTON_BRIDGE_HOST,
    port: PROTON_BRIDGE_PORT,
    tls: false,
    tlsOptions: { rejectUnauthorized: false },
    keepalive: true,
  });

  async function sendNotification(from: string, subject: string) {
    if (!(await hasValidAccount())) {
      logVerbose('Skipping notification (Signal not linked)');
      return;
    }

    try {
      const groupId = await getOrCreateGroup(`proton-${SUP_TOPIC}`, SUP_TOPIC);

      await sendGroupMessage(groupId, subject, {
        androidPackage: 'ch.protonmail.android',
        title: from,
      });

      logVerbose(`Email from ${from}: ${subject}`);
    } catch (error) {
      logError('Failed to send notification:', error);
    }
  }

  const openInbox = () =>
    imap.openBox('INBOX', false, (err, box) => {
      if (err) {
        logError('Failed to open inbox:', err);
        return;
      }

      imap.on('mail', async (numNewMsgs: number) => {
        logVerbose(`${numNewMsgs} new message(s) received`);

        const fetch = imap.seq.fetch(`${box.messages.total}:*`, {
          bodies: 'HEADER.FIELDS (FROM SUBJECT)',
          struct: true,
        });

        fetch.on('message', (msg) => {
          msg.on('body', (stream) => {
            let buffer = '';
            stream.on('data', (chunk) => {
              buffer += chunk.toString('utf8');
            });
            stream.once('end', () => {
              const header = Imap.parseHeader(buffer);
              const rawFrom = header.from?.[0] || 'Unknown sender';
              const subject = header.subject?.[0] || 'No subject';

              const nameMatch = rawFrom.match(/^"?([^"<]+)"?\s*<?/);
              const from = nameMatch?.[1]?.trim() || rawFrom;

              sendNotification(from, subject);
            });
          });
        });
      });
    });

  imap.on('ready', () => {
    imapConnected = true;
    logSuccess('IMAP is ready');
    openInbox();
  });

  imap.on('error', (err: Error) => {
    imapConnected = false;
    logError('IMAP error:', err.message);
  });

  imap.on('close', (hadError: boolean) => {
    imapConnected = false;
    logError(`IMAP connection closed (hadError: ${hadError})`);
  });

  imap.on('end', () => {
    imapConnected = false;
    logError('IMAP connection ended by server');
  });

  imap.connect();

  process.on('SIGTERM', () => imap.end());

  process.on('SIGINT', () => imap.end());
}
