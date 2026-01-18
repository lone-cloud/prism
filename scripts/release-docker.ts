import { $ } from 'bun';

const registry = 'ghcr.io/lone-cloud';
const config = { name: 'sup-server', path: './server' };

try {
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
