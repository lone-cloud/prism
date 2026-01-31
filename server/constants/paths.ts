const HOME = process.env.HOME || '/root';

const SIGNAL_CLI_BIN = `${import.meta.dir}/../../signal-cli/bin/signal-cli`;
export const SIGNAL_CLI = (await Bun.file(SIGNAL_CLI_BIN).exists()) ? SIGNAL_CLI_BIN : 'signal-cli';

export const SIGNAL_CLI_SOCKET = '/tmp/signal-cli.sock';
export const SIGNAL_CLI_DATA_DIR = `${HOME}/.local/share/signal-cli`;
export const SIGNAL_CLI_DATA = `${HOME}/.local/share/signal-cli/data`;

export const PRISM_DB = `${HOME}/.local/share/prism/store.db`;

const PUBLIC_DIR_LOCAL = `${import.meta.dir}/../public`;
export const PUBLIC_DIR = (await Bun.file(`${PUBLIC_DIR_LOCAL}/favicon.webp`).exists())
  ? PUBLIC_DIR_LOCAL
  : '/public';
