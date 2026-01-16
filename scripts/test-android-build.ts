import { join } from 'node:path';
import { $ } from 'bun';

const ANDROID_DIR = join(import.meta.dir, '..', 'android');
const APK_PATH = join(ANDROID_DIR, 'app/build/outputs/apk/release/app-release.apk');

console.log('Building signed release APK...\n');

const requiredEnvVars = ['KEYSTORE_FILE', 'KEYSTORE_PASSWORD', 'KEY_ALIAS', 'KEY_PASSWORD'];
const missing = requiredEnvVars.filter((v) => !process.env[v]);
if (missing.length > 0) {
  console.error(`❌ Missing required environment variables: ${missing.join(', ')}`);
  console.error('\nSee android/RELEASE.md for setup instructions');
  process.exit(1);
}

try {
  process.chdir(ANDROID_DIR);

  console.log('Building release APK...');
  await $`./gradlew assembleRelease`;

  const apkExists = await Bun.file(APK_PATH).exists();
  if (!apkExists) {
    console.error(`\n❌ APK not found at ${APK_PATH}`);
    process.exit(1);
  }

  console.log(`\n✓ APK built successfully: ${APK_PATH}`);
} catch (error) {
  console.error('Build failed:', error);
  process.exit(1);
}
