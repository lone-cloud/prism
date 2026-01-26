import { $ } from 'bun';

try {
  const packageJson = await Bun.file(`package.json`).json();
  const version = `v${packageJson.version}`;

  console.log(`Triggering release ${version} via CI...`);

  const existingTags = await $`git tag -l ${version}`.text();
  if (existingTags.trim()) {
    console.error(`\n❌ Tag ${version} already exists!`);
    console.log(`\nTo re-release, delete the tag first:`);
    console.log(`  git tag -d ${version}`);
    console.log(`  git push origin :refs/tags/${version}`);
    process.exit(1);
  }

  console.log(`\nCreating git tag ${version}...`);
  await $`git tag -a ${version} -m "Release ${version}"`;

  console.log(`Pushing tag to trigger CI build...`);
  await $`git push origin ${version}`;

  console.log(`
✨ Release ${version} triggered!

GitHub Actions will now:
  1. Build Docker images
  2. Push to ghcr.io/lone-cloud/sup-server:${version}
  3. Push to ghcr.io/lone-cloud/sup-server:latest

Watch the build:
  https://github.com/lone-cloud/sup/actions

Once complete, images will be available:
  docker pull ghcr.io/lone-cloud/sup-server:${version}
`);
} catch (error) {
  console.error('Release failed:', error);
  process.exit(1);
}
