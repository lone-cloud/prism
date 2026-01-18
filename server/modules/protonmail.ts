import Imap from 'imap';
import {
  BRIDGE_IMAP_PASSWORD,
  BRIDGE_IMAP_USERNAME,
  ENABLE_PROTON_ANDROID,
  LAUNCH_ENDPOINT_PREFIX,
  PROTON_BRIDGE_HOST,
  PROTON_BRIDGE_PORT,
  SUP_TOPIC,
} from '../constants/config';
import { logError, logInfo, logSuccess, logVerbose, logWarn } from '../utils/log';
import { createGroup, sendGroupMessage } from './signal';
import { getGroupId, register } from './store';

export async function startProtonMonitor() {
  if (!BRIDGE_IMAP_USERNAME || !BRIDGE_IMAP_PASSWORD) {
    logError('Missing required env vars: BRIDGE_IMAP_USERNAME and BRIDGE_IMAP_PASSWORD');
    logWarn('Run: docker compose run --rm protonmail-bridge init');
    logWarn('Then use `login` and `info` commands to get IMAP credentials');
    return;
  }

  logInfo(`🔗 Connecting to Proton Bridge at ${PROTON_BRIDGE_HOST}:${PROTON_BRIDGE_PORT}`);
  logInfo(`📨 Monitoring mailbox: ${BRIDGE_IMAP_USERNAME}`);

  let imap: Imap;
  try {
    imap = new Imap({
      user: BRIDGE_IMAP_USERNAME,
      password: BRIDGE_IMAP_PASSWORD,
      host: PROTON_BRIDGE_HOST,
      port: PROTON_BRIDGE_PORT,
      tls: true,
      tlsOptions: { rejectUnauthorized: false },
      keepalive: true,
    });
  } catch (err) {
    logError('❌ Failed to initialize IMAP client:', err);
    logWarn('⚠️  ProtonMail integration disabled (bridge not reachable)');
    return;
  }

  async function sendNotification(title: string, message: string) {
    try {
      const topicKey = `proton-${SUP_TOPIC}`;
      const groupId = getGroupId(topicKey) ?? (await createGroup(SUP_TOPIC));

      if (!getGroupId(topicKey)) {
        register(topicKey, groupId, SUP_TOPIC);
      }

      const prefix = ENABLE_PROTON_ANDROID
        ? `${LAUNCH_ENDPOINT_PREFIX}ch.protonmail.android]\n`
        : '';
      await sendGroupMessage(groupId, `${prefix}**${title}**\n${message}`);

      logSuccess(`✅ Notification sent: ${title}`);
    } catch (error) {
      logError('❌ Failed to send notification:', error);
    }
  }

  function openInbox() {
    imap.openBox('INBOX', false, (err, box) => {
      if (err) {
        logError('Failed to open inbox:', err);
        return;
      }

      logVerbose(`✅ Connected to inbox (${box.messages.total} messages)`);

      imap.on('mail', async (numNewMsgs: number) => {
        logVerbose(`📬 ${numNewMsgs} new message(s) received`);

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
              const from = header.from?.[0] || 'Unknown sender';
              const subject = header.subject?.[0] || 'No subject';

              sendNotification(`New Mail from ${from}`, subject);
            });
          });
        });
      });

      imap.on('update', () => {
        logVerbose('📊 Mailbox updated');
      });
    });
  }

  imap.once('ready', () => {
    logVerbose('✅ IMAP connection ready');
    openInbox();
  });

  imap.once('error', (err: Error) => {
    logError('❌ IMAP error:', err);
    logWarn('⚠️  ProtonMail integration disabled due to connection error');
  });

  imap.once('end', () => {
    logVerbose('⚠️ IMAP connection ended, reconnecting...');
    setTimeout(() => {
      try {
        imap.connect();
      } catch (err) {
        logError('❌ Failed to reconnect:', err);
      }
    }, 5000);
  });

  try {
    imap.connect();
  } catch (err) {
    logError('❌ Failed to connect to Proton Bridge:', err);
    logWarn('⚠️  ProtonMail integration disabled');
  }

  process.on('SIGTERM', () => {
    imap.end();
  });

  process.on('SIGINT', () => {
    imap.end();
  });
}
