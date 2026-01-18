import { rm } from 'node:fs/promises';
import chalk from 'chalk';
import { DAEMON_START_MAX_ATTEMPTS, DEVICE_NAME, VERBOSE } from '../constants/config';
import { SIGNAL_CLI, SIGNAL_CLI_DATA, SIGNAL_CLI_SOCKET } from '../constants/paths';
import type { ListAccountsResult, StartLinkResult, UpdateGroupResult } from '../types';
import { log } from '../utils/log';

log(`Running signal-cli from ${SIGNAL_CLI}`);

const MESSAGE_DELIMITER = '\n';
let account: string | null = null;
let currentLinkUri: string | null = null;
let rpcId = 1;

export async function initSignal({ accountOverride }: { accountOverride?: string }) {
  if (accountOverride) {
    account = accountOverride;
    return true;
  }

  const result = (await rpcCall('listAccounts', {})) as ListAccountsResult;
  const [firstAccount] = result;
  if (firstAccount) {
    account = firstAccount;
    return true;
  }

  return false;
}

async function rpcCall(method: string, params: Record<string, unknown>) {
  return new Promise((resolve, reject) => {
    let response = '';

    Bun.connect({
      unix: SIGNAL_CLI_SOCKET,
      socket: {
        data(socket, data) {
          response += new TextDecoder().decode(data);

          const isComplete = response.includes(MESSAGE_DELIMITER);
          if (isComplete) {
            socket.end();

            const parsed = JSON.parse(response.trim());

            if (parsed.error) {
              reject(new Error(`signal-cli RPC error: ${parsed.error.message}`));
            } else {
              resolve(parsed.result);
            }
          }
        },
        error(_socket, error) {
          reject(error);
        },
        close() {
          const isComplete = response.includes(MESSAGE_DELIMITER);
          if (!isComplete) {
            reject(new Error('Connection closed before response received'));
          }
        },
        open(socket) {
          const request = JSON.stringify({
            jsonrpc: '2.0',
            id: rpcId++,
            method,
            params: account ? { account, ...params } : params,
          });

          socket.write(`${request}${MESSAGE_DELIMITER}`);
        },
      },
    });
  });
}

export async function generateLinkQR() {
  const result = (await rpcCall('startLink', {
    deviceName: DEVICE_NAME,
  })) as StartLinkResult;
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

  const result = await rpcCall('finishLink', {
    deviceLinkUri: currentLinkUri,
    deviceName: DEVICE_NAME,
  });
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
  const result = (await rpcCall('updateGroup', {
    name,
    member: members,
  })) as UpdateGroupResult;

  if (!result?.groupId) {
    throw new Error('Failed to create group');
  }

  return result.groupId;
}

export async function sendGroupMessage(groupId: string, message: string) {
  await rpcCall('send', {
    groupId,
    message,
  });
}

export async function checkSignalCli() {
  try {
    await rpcCall('listAccounts', {});
    return true;
  } catch {
    return false;
  }
}

export async function hasValidAccount() {
  try {
    const result = (await rpcCall('listAccounts', {})) as ListAccountsResult;
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
          console.error(chalk.red('[signal-cli]'), trimmed);
        } else if (trimmed.includes('WARN')) {
          console.warn(chalk.yellow('[signal-cli]'), trimmed);
        } else {
          console.log(chalk.dim('[signal-cli]'), trimmed);
        }
        continue;
      }

      if (trimmed.includes('WARN')) {
        console.warn(chalk.yellow('[signal-cli]'), trimmed);
      } else if (trimmed.includes('ERROR') || !trimmed.includes('INFO')) {
        console.error(chalk.red('[signal-cli]'), trimmed);
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
      console.log(chalk.green('✓ signal-cli daemon started'));
      return proc;
    } catch (_error) {
      if (authError && attempts > 5 && !cleaned) {
        console.log(chalk.yellow('⚠ Detected stale account data, cleaning up and retrying...'));
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
