import { rm, unlink } from 'node:fs/promises';
import { DEVICE_NAME, PORT, VERBOSE } from '@/constants/config';
import { SIGNAL_CLI, SIGNAL_CLI_DATA, SIGNAL_CLI_SOCKET } from '@/constants/paths';
import type { ListAccountsResult, StartLinkResult, UpdateGroupResult } from '@/types';
import { logError, logInfo, logSuccess, logVerbose, logWarn } from '@/utils/log';
import { call } from '@/utils/rpc';

let account: string | null = null;
let currentLinkUri: string | null = null;
let daemon: ReturnType<typeof Bun.spawn> | null = null;

export const hasLinkUri = () => currentLinkUri !== null;

export async function initSignal({ accountOverride }: { accountOverride?: string } = {}) {
  await startDaemon();

  const isLinked = await checkSignalCli();

  if (!isLinked) {
    logWarn('No Signal account linked');
    logInfo(`Visit http://localhost:${PORT} to link your device`);
    return { linked: false, account: null };
  }

  if (accountOverride) {
    account = accountOverride;
    logVerbose(`Signal account set: ${account}`);
    logSuccess('Signal account linked');
    return { linked: true, account };
  }

  const result = (await call('listAccounts', {}, null)) as ListAccountsResult;
  const [firstAccount] = result;
  if (firstAccount) {
    account = firstAccount.number;
    logVerbose(`Signal account initialized: ${account}`);
    logSuccess('Signal account linked');
    return { linked: true, account };
  }

  logVerbose('No Signal accounts found');
  logWarn('No Signal account linked');
  logInfo(`Visit http://localhost:${PORT} to link your device`);
  return { linked: false, account: null };
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

  const uri = currentLinkUri;
  currentLinkUri = null;

  const result = await call(
    'finishLink',
    {
      deviceLinkUri: uri,
      deviceName: DEVICE_NAME,
    },
    account,
  );
  logSuccess('Device linked successfully');
  return result;
}

export async function unlinkDevice() {
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
  { notifySelf = true }: { notifySelf?: boolean } = {},
) {
  await call(
    'send',
    {
      groupId,
      message,
      'notify-self': notifySelf,
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

  try {
    await unlink(SIGNAL_CLI_SOCKET);
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

  await Bun.sleep(3000);

  try {
    const socket = await Bun.connect({
      unix: SIGNAL_CLI_SOCKET,
      socket: {
        data() {},
      },
    });
    socket.end();
    logSuccess('signal-cli daemon started');
    daemon = proc;
    return proc;
  } catch (error) {
    if (authError && !cleaned) {
      logWarn(' Detected stale account data, cleaning up and retrying...');
      proc.kill();
      await unlinkDevice();
      cleaned = true;
      return startDaemon();
    }

    logError('Failed to connect to signal-cli socket:', error);
    if (proc.exitCode !== null) {
      logError('signal-cli process exited with code:', proc.exitCode);
    }
    throw new Error('Failed to start signal-cli daemon');
  }
}

export async function restartDaemon() {
  if (daemon) {
    daemon.kill();
    await Bun.sleep(500);
  }
  daemon = await startDaemon();
}

export const cleanupDaemon = () => daemon?.kill();
