import { join } from 'node:path';
import { $ } from 'bun';

const ANDROID_DIR = join(import.meta.dir, '..', 'android');
const APK_PATH = join(ANDROID_DIR, 'app/build/outputs/apk/release/app-release.apk');

async function main() {
  console.log('Building signed release APK...\n');

  const requiredEnvVars = ['KEYSTORE_FILE', 'KEYSTORE_PASSWORD', 'KEY_ALIAS', 'KEY_PASSWORD'];
  const missing = requiredEnvVars.filter((v) => !process.env[v]);
  if (missing.length > 0) {
    console.error(`❌ Missing required environment variables: ${missing.join(', ')}`);
    console.error('\nSee android/RELEASE.md for setup instructions');
    process.exit(1);
  }

  process.chdir(ANDROID_DIR);

  console.log('Building release APK...');
  await $`./gradlew assembleRelease`;

  const apkExists = await Bun.file(APK_PATH).exists();
  if (!apkExists) {
    console.error(`\n❌ APK not found at ${APK_PATH}`);
    process.exit(1);
  }

  console.log(`\n✓ Built: ${APK_PATH}`);

  const sha256Output = await $`sha256sum ${APK_PATH}`.text();
  const sha256 = sha256Output.split(/\s+/)[0];
  console.log(`✓ SHA256: ${sha256}`);

  const certOutput =
    await $`unzip -p ${APK_PATH} META-INF/*.RSA | keytool -printcert | grep "SHA256:" | awk '{print $2}'`.text();
  const fingerprint = certOutput.trim();
  console.log(`✓ Certificate SHA256: ${fingerprint}`);
}

main().catch((error) => {
  console.error('Build failed:', error.message);
  process.exit(1);
});
