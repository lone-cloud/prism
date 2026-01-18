import { rm } from 'node:fs/promises';
import { DAEMON_START_MAX_ATTEMPTS, DEVICE_NAME, VERBOSE } from '../constants/config';
import { SIGNAL_CLI, SIGNAL_CLI_DATA, SIGNAL_CLI_SOCKET } from '../constants/paths';
import type { ListAccountsResult, StartLinkResult, UpdateGroupResult } from '../types';
import { logError, logInfo, logSuccess, logVerbose, logWarn } from '../utils/log';
import { call } from '../utils/rpc';

logVerbose(`Running signal-cli from ${SIGNAL_CLI}`);

let account: string | null = null;
let currentLinkUri: string | null = null;

export async function initSignal({ accountOverride }: { accountOverride?: string }) {
  if (accountOverride) {
    account = accountOverride;
    return true;
  }

  const result = (await call('listAccounts', {}, account)) as ListAccountsResult;
  const [firstAccount] = result;
  if (firstAccount) {
    account = firstAccount;
    return true;
  }

  return false;
}

export async function generateLinkQR() {
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

  currentLinkUri = uri;
  return `https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=${encodeURIComponent(uri)}`;
}

export async function finishLink() {
  if (!currentLinkUri) {
    throw new Error('No link in progress');
  }

  const result = await call(
    'finishLink',
    {
      deviceLinkUri: currentLinkUri,
      deviceName: DEVICE_NAME,
    },
    account,
  );
  currentLinkUri = null;
  return result;
}

export async function unlinkDevice() {
  account = null;
  currentLinkUri = null;

  try {
    await rm(SIGNAL_CLI_DATA, { recursive: true, force: true });
  } catch {}
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

export async function sendGroupMessage(groupId: string, message: string) {
  await call(
    'send',
    {
      groupId,
      message,
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

      if (VERBOSE) {
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

  let attempts = 0;
  while (attempts < DAEMON_START_MAX_ATTEMPTS) {
    try {
      const socket = await Bun.connect({
        unix: SIGNAL_CLI_SOCKET,
        socket: {
          data() {},
          error() {},
        },
      });
      socket.end();
      logSuccess('✓ signal-cli daemon started');
      return proc;
    } catch (_error) {
      if (authError && attempts > 5 && !cleaned) {
        logWarn('⚠ Detected stale account data, cleaning up and retrying...');
        proc.kill();
        await unlinkDevice();
        cleaned = true;
        return startDaemon();
      }

      await new Promise((resolve) => setTimeout(resolve, 100));
      attempts++;
    }
  }

  throw new Error('Failed to start signal-cli daemon');
}
