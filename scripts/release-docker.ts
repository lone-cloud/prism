import { $ } from 'bun';

const service = process.argv[2];

if (!service || !['server', 'bridge'].includes(service)) {
  console.error('Usage: bun run docker:release <service>');
  console.error('Example: bun run docker:release server');
  console.error('         bun run docker:release bridge');
  process.exit(1);
}

const registry = 'ghcr.io/lone-cloud';
const config =
  service === 'server'
    ? { name: 'sup-server', path: './server' }
    : { name: 'sup-proton-bridge', path: './proton-bridge' };

try {
  // Read version from package.json
  const packageJson = await Bun.file(`${config.path}/package.json`).json();
  const version = `v${packageJson.version}`;

  console.log(`🚀 Releasing ${config.name} ${version}...`);

  const fullName = `${registry}/${config.name}`;

  console.log(`\n📦 Building ${config.name}...`);
  await $`docker build -t ${fullName}:${version} -t ${fullName}:latest ${config.path}`;
  console.log(`✓ Built ${config.name}`);

  console.log(`📤 Pushing ${fullName}:${version}...`);
  await $`docker push ${fullName}:${version}`;
  console.log(`✓ Pushed ${fullName}:${version}`);

  console.log(`📤 Pushing ${fullName}:latest...`);
  await $`docker push ${fullName}:latest`;
  console.log(`✓ Pushed ${fullName}:latest`);

  console.log(`
✨ ${config.name} ${version} released successfully!

Images pushed:
  - ${fullName}:${version}
  - ${fullName}:latest

Users can now pull with:
  docker compose pull
`);
} catch (error) {
  console.error('❌ Release failed:', error);
  process.exit(1);
}
