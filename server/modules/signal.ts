import { rm } from 'node:fs/promises';
import {
  DEVICE_NAME,
  ENABLE_ANDROID_INTEGRATION,
  LAUNCH_ENDPOINT_PREFIX,
  VERBOSE_LOGGING,
} from '@/constants/config';
import { SIGNAL_CLI, SIGNAL_CLI_DATA, SIGNAL_CLI_SOCKET } from '@/constants/paths';
import { formatPhoneNumber } from '@/utils/format';
import { logError, logInfo, logSuccess, logVerbose, logWarn } from '@/utils/log';
import { call } from '@/utils/rpc';

interface StartLinkResult {
  deviceLinkUri: string;
}

interface UpdateGroupResult {
  groupId: string;
}

type ListAccountsResult = {
  number: string;
  name?: string;
  uuid?: string;
}[];

let account: string | null = null;
let currentLinkUri: string | null = null;
let daemon: ReturnType<typeof Bun.spawn> | null = null;

export async function initSignal({ accountOverride }: { accountOverride?: string } = {}) {
  await startDaemon();

  const isLinked = await checkSignalCli();

  if (!isLinked) {
    return { linked: false, account: null };
  }

  if (accountOverride) {
    account = accountOverride;
    logVerbose(`Signal account set: ${account}`);

    return { linked: true, account };
  }

  const result = (await call('listAccounts', {}, null)) as ListAccountsResult;
  const [firstAccount] = result;

  if (firstAccount) {
    account = firstAccount.number;
    logVerbose(`Signal account initialized: ${formatPhoneNumber(account)}`);
    return { linked: true, account };
  }

  logVerbose('No Signal accounts found');

  return { linked: false, account: null };
}

export async function generateLinkQR() {
  try {
    const result = (await call(
      'startLink',
      {
        deviceName: DEVICE_NAME,
      },
      account,
    )) as StartLinkResult;
    const uri = result.deviceLinkUri;

    if (!uri) {
      throw new Error('Failed to generate linking URI');
    }

    logVerbose(`Generated link URI: ${uri.substring(0, 30)}...`);
    currentLinkUri = uri;

    const qrUrl = `https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=${encodeURIComponent(uri)}`;
    const response = await fetch(qrUrl);
    const buffer = await response.arrayBuffer();
    const base64 = Buffer.from(buffer).toString('base64');

    return `data:image/png;base64,${base64}`;
  } catch (error) {
    logError('Failed to generate link QR:', error);
    throw error;
  }
}

export async function finishLink() {
  if (!currentLinkUri) {
    throw new Error('No link in progress');
  }

  const uri = currentLinkUri;
  currentLinkUri = null;

  const result = await call(
    'finishLink',
    {
      deviceLinkUri: uri,
      deviceName: DEVICE_NAME,
    },
    null,
  );

  logSuccess('Device linked successfully');

  return result;
}

async function unlinkDevice() {
  account = null;
  currentLinkUri = null;

  if (await Bun.file(SIGNAL_CLI_DATA).exists()) {
    logWarn('Removing local account data...');

    try {
      await rm(SIGNAL_CLI_DATA, { recursive: true, force: true });

      logSuccess('Account data removed');
    } catch (error) {
      logError('Failed to remove account data directory:', error);
    }
  }
}

export async function createGroup(name: string, members: string[] = []) {
  const result = (await call(
    'updateGroup',
    {
      name,
      member: members,
    },
    account,
  )) as UpdateGroupResult;

  if (!result?.groupId) {
    throw new Error('Failed to create group');
  }

  return result.groupId;
}

export async function sendGroupMessage(
  groupId: string,
  message: string,
  options?: { androidPackage?: string; title?: string },
) {
  let formattedMessage = message;

  if (ENABLE_ANDROID_INTEGRATION && options?.androidPackage) {
    const title = options.title ? `**${options.title}**\n` : '';
    formattedMessage = `${LAUNCH_ENDPOINT_PREFIX}${options.androidPackage}]\n${title}${message}`;
  } else if (options?.title) {
    formattedMessage = `${options.title}\n${message}`;
  }

  await call(
    'send',
    {
      groupId,
      message: formattedMessage,
      'notify-self': true,
    },
    account,
  );
}

export async function checkSignalCli() {
  try {
    await call('listAccounts', {}, account);

    return true;
  } catch {
    return false;
  }
}

export async function hasValidAccount() {
  try {
    const result = (await call('listAccounts', {}, account)) as ListAccountsResult;

    return result.length > 0;
  } catch {
    return false;
  }
}

export async function startDaemon() {
  let authError = false;
  let cleaned = false;

  if (daemon && !daemon.killed) {
    daemon.kill();
    await Bun.sleep(2000);
  }

  try {
    await Bun.file(SIGNAL_CLI_SOCKET).delete();
    logVerbose('Removed stale socket file');
  } catch (error) {
    if (error instanceof Error && 'code' in error && error.code !== 'ENOENT') {
      logError('Failed to remove stale socket file:', error);
    }
  }

  const proc = Bun.spawn([SIGNAL_CLI, 'daemon', '--socket', SIGNAL_CLI_SOCKET], {
    stdout: 'pipe',
    stderr: 'pipe',
  });

  (async () => {
    for await (const chunk of proc.stderr) {
      const text = new TextDecoder().decode(chunk);
      const trimmed = text.trim();

      if (!trimmed) continue;

      if (trimmed.includes('Authorization failed') || trimmed.includes('AccountCheckException')) {
        authError = true;
      }

      if (trimmed.includes('ConcurrentModificationException')) continue;

      if (
        trimmed.includes('Accepted new client connection') ||
        (trimmed.includes('Connection') && trimmed.includes('closed'))
      )
        continue;

      if (VERBOSE_LOGGING) {
        if (trimmed.includes('ERROR')) {
          logError('[signal-cli]', trimmed);
        } else if (trimmed.includes('WARN')) {
          logWarn('[signal-cli]', trimmed);
        } else {
          logInfo('[signal-cli]', trimmed);
        }
        continue;
      }

      if (trimmed.includes('WARN')) {
        logWarn('[signal-cli]', trimmed);
      } else if (trimmed.includes('ERROR') || !trimmed.includes('INFO')) {
        logError('[signal-cli]', trimmed);
      }
    }
  })();

  for (let i = 0; i < 60; i++) {
    await Bun.sleep(500);

    try {
      const socket = await Bun.connect({
        unix: SIGNAL_CLI_SOCKET,
        socket: {
          data() {},
        },
      });

      socket.end();

      logSuccess(`signal-cli daemon started (${(i + 1) * 0.5}s)`);

      daemon = proc;

      return proc;
    } catch {}
  }

  if (authError && !cleaned) {
    logWarn('Detected stale account data, cleaning up and retrying...');
    
    proc.kill();

    await unlinkDevice();

    cleaned = true;

    return startDaemon();
  }

  logError('Failed to connect to signal-cli socket: daemon did not start within 30 seconds');

  if (proc.exitCode !== null) {
    logError('signal-cli process exited with code:', proc.exitCode);
  }

  if (authError) {
    logError('Account authorization failed. You may need to unlink and re-link your device.');
  }

  throw new Error('Failed to start signal-cli daemon');
}

export const cleanupDaemon = () => daemon?.kill();

export const getAccount = () => account;
