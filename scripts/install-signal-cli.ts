import { chmod, rename } from 'node:fs/promises';

const SIGNAL_CLI_VERSION = '0.13.22';
const SIGNAL_CLI_URL = `https://github.com/AsamK/signal-cli/releases/download/v${SIGNAL_CLI_VERSION}/signal-cli-${SIGNAL_CLI_VERSION}.tar.gz`;
const SIGNAL_CLI_DIR = `${import.meta.dir}/../signal-cli`;

async function installSignalCli() {
  if (await Bun.file(`${SIGNAL_CLI_DIR}/bin/signal-cli`).exists()) {
    return;
  }

  console.log('Downloading signal-cli...');

  const response = await fetch(SIGNAL_CLI_URL);
  if (!response.ok) {
    throw new Error(`Failed to download: ${response.statusText}`);
  }

  console.log('Extracting signal-cli...');

  const archive = new Bun.Archive(await response.blob());
  await archive.extract(`${import.meta.dir}/..`);

  const extractedDir = `${import.meta.dir}/../signal-cli-${SIGNAL_CLI_VERSION}`;
  await rename(extractedDir, SIGNAL_CLI_DIR);

  await chmod(`${SIGNAL_CLI_DIR}/bin/signal-cli`, 0o755);

  console.log('signal-cli installed successfully');
}

await installSignalCli();
