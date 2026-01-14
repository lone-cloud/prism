import { $ } from 'bun';

async function main() {
  const version = process.argv[2];

  if (!version) {
    console.error('Usage: bun run release <version>');
    console.error('Example: bun run release v0.1.0');
    process.exit(1);
  }

  if (!version.startsWith('v')) {
    console.error('❌ Version must start with "v" (e.g., v0.1.0)');
    process.exit(1);
  }

  const status = await $`git status --porcelain`.text();
  if (status.trim()) {
    console.error('❌ You have uncommitted changes. Commit or stash them first.');
    process.exit(1);
  }

  try {
    await $`git rev-parse ${version}`.quiet();
    console.error(`❌ Tag ${version} already exists`);
    process.exit(1);
  } catch {
    // Tag doesn't exist, good
  }

  console.log(`🚀 Creating release ${version}...`);

  await $`git tag ${version}`;
  console.log(`✓ Created tag ${version}`);

  await $`git push origin ${version}`;
  console.log(`✓ Pushed tag to GitHub`);

  console.log(`
✓ Release ${version} triggered!

GitHub Actions will now:
1. Build signed Android APK
2. Create GitHub release
3. Upload APK with SHA256 hash

Check progress: https://github.com/lone-cloud/sup/actions
`);
}

main().catch((error) => {
  console.error('Release failed:', error.message);
  process.exit(1);
});
