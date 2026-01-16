import { $ } from 'bun';

console.log('Regenerating Android dependency lockfiles...');

try {
  await $`cd android && ./gradlew dependencies --write-locks`;
  console.log('✓ Lockfiles regenerated successfully');
} catch (error) {
  console.error('✗ Failed to regenerate lockfiles');
  console.error(error);
  process.exit(1);
}
