# SUP Android App

UnifiedPush distributor that routes notifications through Signal.

## Setup for Development in VS Code

### Prerequisites

1. **Install Android SDK** (via Android Studio or command line tools)
2. **Install Java 17+**: `sudo pacman -S jdk17-openjdk`
3. **Set environment variables** in `~/.zshrc`:
   ```bash
   export ANDROID_HOME=$HOME/Android/Sdk
   export PATH=$PATH:$ANDROID_HOME/platform-tools:$ANDROID_HOME/cmdline-tools/latest/bin
   ```

### Generate Gradle Wrapper (First Time)

```bash
cd android

# Download gradle temporarily
wget https://services.gradle.org/distributions/gradle-8.2-bin.zip
unzip gradle-8.2-bin.zip
./gradle-8.2/bin/gradle wrapper --gradle-version 8.2
rm -rf gradle-8.2 gradle-8.2-bin.zip

# Make wrapper executable
chmod +x gradlew
```

### Build in VS Code

Use **Ctrl+Shift+B** or run tasks from Command Palette:
- `Android: Build Debug APK`
- `Android: Install Debug APK`
- `Android: View Logs (Logcat)`

### Manual Commands

```bash
cd android
./gradlew assembleDebug      # Build
./gradlew installDebug        # Install
adb logcat -s SUP:*           # View logs
```


```bash
cd android
# Use Android Studio or command line to create new Android project
```
