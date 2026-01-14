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

  const tarball = await response.arrayBuffer();

  console.log('Extracting signal-cli...');

  const proc = Bun.spawn(['tar', 'xzf', '-'], {
    stdin: 'pipe',
    cwd: `${import.meta.dir}/..`,
  });

  proc.stdin.write(new Uint8Array(tarball));
  proc.stdin.end();
  await proc.exited;

  const extractedDir = `${import.meta.dir}/../signal-cli-${SIGNAL_CLI_VERSION}`;
  await Bun.spawn(['mv', extractedDir, SIGNAL_CLI_DIR]).exited;

  await Bun.spawn(['chmod', '+x', `${SIGNAL_CLI_DIR}/bin/signal-cli`]).exited;

  console.log('✓ signal-cli installed successfully');
}

await installSignalCli();
