import { chmod, rename, rm } from 'node:fs/promises';

const SIGNAL_CLI_VERSION = process.env.SIGNAL_CLI_VERSION || '0.13.23';
const SIGNAL_CLI_URL = `https://github.com/AsamK/signal-cli/releases/download/v${SIGNAL_CLI_VERSION}/signal-cli-${SIGNAL_CLI_VERSION}.tar.gz`;
const SIGNAL_CLI_DIR = `${import.meta.dir}/../signal-cli`;

async function installSignalCli() {
  const binaryPath = `${SIGNAL_CLI_DIR}/bin/signal-cli`;
  const binaryExists = await Bun.file(binaryPath).exists();

  if (binaryExists) {
    try {
      const proc = Bun.spawn([binaryPath, '--version'], { stdout: 'pipe' });
      const output = await new Response(proc.stdout).text();
      const installedVersion = output.trim().replace(/^signal-cli\s+/, '');

      if (installedVersion === SIGNAL_CLI_VERSION) {
        return;
      }

      console.log(`Upgrading signal-cli from ${installedVersion} to ${SIGNAL_CLI_VERSION}...`);
      await rm(SIGNAL_CLI_DIR, { recursive: true, force: true });
    } catch {
      console.log('Reinstalling signal-cli (version check failed)...');
      await rm(SIGNAL_CLI_DIR, { recursive: true, force: true });
    }
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
