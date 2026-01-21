const HOME = process.env.HOME || '/root';

const SIGNAL_CLI_BIN = '../signal-cli/bin/signal-cli';
export const SIGNAL_CLI = (await Bun.file(SIGNAL_CLI_BIN).exists()) ? SIGNAL_CLI_BIN : 'signal-cli';

export const SIGNAL_CLI_SOCKET = '/tmp/signal-cli.sock';
export const SIGNAL_CLI_DATA_DIR = `${HOME}/.local/share/signal-cli`;
export const SIGNAL_CLI_DATA = `${HOME}/.local/share/signal-cli/data`;

export const SUP_DB = `${HOME}/.local/share/sup/store.db`;

export const PUBLIC_DIR = (await Bun.file('public/favicon.webp').exists()) ? 'public' : '/public';
