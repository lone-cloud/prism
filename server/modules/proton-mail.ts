import Imap from 'imap';
import {
  PROTON_BRIDGE_HOST,
  PROTON_BRIDGE_PORT,
  PROTON_IMAP_PASSWORD,
  PROTON_IMAP_USERNAME,
  PROTON_SUP_TOPIC,
} from '@/constants/config';
import { hasValidAccount, sendGroupMessage } from '@/modules/signal';
import { getOrCreateGroup } from '@/modules/store';
import { logError, logInfo, logSuccess, logVerbose, logWarn } from '@/utils/log';

let imapConnected = false;
let monitorStartTime = 0;
let reconnectAttempts = 0;
const MAX_RECONNECT_DELAY = 300000; // 5 minutes
let imapInstance: Imap | null = null;

export const isImapConnected = () => imapConnected;

const getReconnectDelay = () => {
  const baseDelay = 10000; // 10 seconds
  const delay = Math.min(baseDelay * 2 ** reconnectAttempts, MAX_RECONNECT_DELAY);
  reconnectAttempts++;
  return delay;
};

async function sendNotification(from: string, subject: string) {
  if (!(await hasValidAccount())) {
    logVerbose('Skipping notification (Signal not linked)');

    return;
  }

  try {
    const groupId = await getOrCreateGroup(`proton-${PROTON_SUP_TOPIC}`, PROTON_SUP_TOPIC);

    await sendGroupMessage(groupId, subject, {
      title: from,
    });

    logVerbose(`Email from ${from}: ${subject}`);
  } catch (error) {
    logError('Failed to send notification:', error);
  }
}

export async function startProtonMonitor() {
  if (!PROTON_IMAP_USERNAME || !PROTON_IMAP_PASSWORD) {
    logError('Missing required env vars: PROTON_IMAP_USERNAME and PROTON_IMAP_PASSWORD');
    logWarn('Run: docker compose run --rm protonmail-bridge init');
    logWarn('Then use `login` and `info` commands to get IMAP credentials');

    return;
  }

  logInfo(`Connecting to Proton Bridge at ${PROTON_BRIDGE_HOST}:${PROTON_BRIDGE_PORT}`);
  logInfo(`Monitoring mailbox: ${PROTON_IMAP_USERNAME}`);

  const imap = new Imap({
    user: PROTON_IMAP_USERNAME,
    password: PROTON_IMAP_PASSWORD,
    host: PROTON_BRIDGE_HOST,
    port: PROTON_BRIDGE_PORT,
    keepalive: true,
  });

  const openInbox = () =>
    imap.openBox('INBOX', false, (err, box) => {
      if (err) {
        logError('Failed to open inbox:', err);

        return;
      }

      if (!monitorStartTime) {
        monitorStartTime = Date.now();
      } else {
        logVerbose('Inbox reopened (reconnection)');
      }

      imap.on('mail', async (numNewMsgs: number) => {
        logVerbose(`${numNewMsgs} new message(s) in mailbox`);

        const fetch = imap.seq.fetch(`${box.messages.total - numNewMsgs + 1}:*`, {
          bodies: 'HEADER.FIELDS (FROM SUBJECT DATE)',
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
              const dateStr = header.date?.[0];

              if (dateStr) {
                const messageDate = new Date(dateStr).getTime();

                if (messageDate < monitorStartTime) {
                  const formattedDate = new Date(dateStr).toLocaleString('en-US', {
                    dateStyle: 'medium',
                    timeStyle: 'short',
                  });

                  logVerbose(`Skipping old email: ${subject} (${formattedDate})`);

                  return;
                }
              }

              const nameMatch = rawFrom.match(/^"?([^"<]+)"?\s*<?/);
              const from = nameMatch?.[1]?.trim() || rawFrom;

              reconnectAttempts = 0;

              sendNotification(from, subject);
            });
          });
        });
      });
    });

  imap.on('ready', () => {
    imapConnected = true;
    reconnectAttempts = 0;
    logSuccess('IMAP is ready');
    openInbox();
  });

  imap.on('error', (err: Error) => {
    imapConnected = false;
    logError('IMAP error:', err.message);
  });

  const handleReconnect = (reason: string) => {
    imapConnected = false;
    logError(reason);

    const delay = getReconnectDelay();
    logInfo(`Attempting to reconnect in ${delay / 1000}s (attempt ${reconnectAttempts})...`);

    setTimeout(() => {
      if (!imapConnected) {
        logInfo('Reconnecting to Proton Bridge...');
        imap.connect();
      }
    }, delay);
  };

  imap.on('close', (hadError: boolean) => {
    handleReconnect(`IMAP connection closed (hadError: ${hadError})`);
  });

  imap.on('end', () => {
    handleReconnect('IMAP connection ended by server');
  });

  imap.connect();

  process.on('SIGTERM', () => imap.end());

  process.on('SIGINT', () => imap.end());

  imapInstance = imap;
}

export async function markEmailAsRead(uid: number) {
  if (!imapInstance || !imapConnected) {
    return { success: false, error: 'IMAP not connected' };
  }

  return new Promise<{ success: boolean; error?: string }>((resolve) => {
    try {
      imapInstance?.addFlags(uid, '\\Seen', (err) => {
        if (err) {
          logError('Failed to mark email as read:', err);
          resolve({ success: false, error: err.message });
        } else {
          resolve({ success: true });
        }
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      resolve({ success: false, error: errorMessage });
    }
  });
}
