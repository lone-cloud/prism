# Android Release Signing

## GitHub Actions Setup (Recommended)

Push a tag to auto-build and release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

**Required GitHub secrets:**
- `KEYSTORE_BASE64`: Run `base64 -w 0 android/release.keystore` and paste output
- `KEYSTORE_PASSWORD`: Your keystore password from `.env`

## Manual Release (Local)

### Generate Signing Key (First Time Only)

```bash
# Generate random passwords
STORE_PASS=$(openssl rand -base64 32)
KEY_PASS=$(openssl rand -base64 32)

# Create keystore (use fake info for CN, OU, O, L, ST, C - it's publicly visible in APKs)
keytool -genkey -v \
  -keystore android/release.keystore \
  -alias sup-release \
  -keyalg RSA \
  -keysize 4096 \
  -validity 10000 \
  -storepass "$STORE_PASS" \
  -keypass "$KEY_PASS"

# Output passwords to save to .env
echo ""
echo "Add to .env (gitignored):"
echo "KEYSTORE_FILE=./android/release.keystore"
echo "KEYSTORE_PASSWORD=$STORE_PASS"
echo "KEY_ALIAS=sup-release"
echo "KEY_PASSWORD=$KEY_PASS"
```

## Get Certificate Fingerprint

For Obtainium/F-Droid verification:

```bash
keytool -list -v \
  -keystore android/release.keystore \
  -alias sup-release \
  | grep "SHA256:"
```

Save this fingerprint - users will verify it in Obtainium to ensure APK authenticity.

## Build Signed Release

```bash
# Load env vars
source .env

# Build
bun run build:android
```

Output will be at: `android/app/build/outputs/apk/release/app-release.apk`

## GitHub Release Process

1. Build signed APK: `bun run build:android`
2. Create GitHub release with tag (e.g., `v0.1.0`)
3. Upload `app-release.apk` 
4. Include SHA256 hash and certificate fingerprint in release notes
5. Users can install via:
   - **Obtainium**: Add repo URL, verify certificate fingerprint
   - **Direct**: Download APK, verify SHA256, install

## Obtainium Setup for Users

1. Install Obtainium from F-Droid
2. Add app: `https://github.com/yourusername/sup`
3. Verify certificate fingerprint matches published value
4. Auto-updates on new GitHub releases

## Reproducible Builds

Dependency lockfile committed: `android/gradle.lockfile`

To update dependencies:
```bash
cd android
./gradlew dependencies --write-locks
git add gradle.lockfile
git commit -m "Update Android dependencies"
```
