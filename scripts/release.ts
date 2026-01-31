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

  console.log(`\nCreating GitHub release...`);
  await $`gh release create ${version} --title ${version} --notes "## Docker Images

Pull the latest version:
\`\`\`bash
docker pull ghcr.io/lone-cloud/prism:${version}
\`\`\`" --generate-notes`;

  console.log(`
✨ Release ${version} complete!

Once CI completes, images will be available:
  docker pull ghcr.io/lone-cloud/prism:${version}
`);
} catch (error) {
  console.error('Release failed:', error);
  process.exit(1);
}
