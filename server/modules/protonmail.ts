import chalk from 'chalk';
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
import { log } from '../utils/log';
import { createGroup, sendGroupMessage } from './signal';
import { getGroupId, register } from './store';

export async function startProtonMonitor() {
  if (!BRIDGE_IMAP_USERNAME || !BRIDGE_IMAP_PASSWORD) {
    console.error(
      chalk.red('Missing required env vars: BRIDGE_IMAP_USERNAME and BRIDGE_IMAP_PASSWORD'),
    );
    console.error(chalk.yellow('Run: docker compose run --rm protonmail-bridge init'));
    console.error(chalk.yellow('Then use `login` and `info` commands to get IMAP credentials'));
    return;
  }

  log(chalk.blue(`🔗 Connecting to Proton Bridge at ${PROTON_BRIDGE_HOST}:${PROTON_BRIDGE_PORT}`));
  log(chalk.blue(`📨 Monitoring mailbox: ${BRIDGE_IMAP_USERNAME}`));

  const imap = new Imap({
    user: BRIDGE_IMAP_USERNAME,
    password: BRIDGE_IMAP_PASSWORD,
    host: PROTON_BRIDGE_HOST,
    port: PROTON_BRIDGE_PORT,
    tls: true,
    tlsOptions: { rejectUnauthorized: false },
    keepalive: true,
  });

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

      console.log(chalk.green(`✅ Notification sent: ${title}`));
    } catch (error) {
      console.error(chalk.red('❌ Failed to send notification:'), error);
    }
  }

  function openInbox() {
    imap.openBox('INBOX', false, (err, box) => {
      if (err) {
        console.error(chalk.red('Failed to open inbox:'), err);
        return;
      }

      log(`✅ Connected to inbox (${box.messages.total} messages)`);

      imap.on('mail', async (numNewMsgs: number) => {
        log(`📬 ${numNewMsgs} new message(s) received`);

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
        log('📊 Mailbox updated');
      });
    });
  }

  imap.once('ready', () => {
    log('✅ IMAP connection ready');
    openInbox();
  });

  imap.once('error', (err: Error) => {
    console.error(chalk.red('❌ IMAP error:'), err);
  });

  imap.once('end', () => {
    log('⚠️ IMAP connection ended, reconnecting...');
    setTimeout(() => imap.connect(), 5000);
  });

  imap.connect();

  process.on('SIGTERM', () => {
    imap.end();
  });

  process.on('SIGINT', () => {
    imap.end();
  });
}
