import { join } from 'node:path';

const BUILD_GRADLE = join(import.meta.dir, '..', 'android', 'app', 'build.gradle.kts');

function isStableVersion(version: string) {
  const unstableKeywords = ['alpha', 'beta', 'rc', 'snapshot', 'dev', 'preview'];
  const lower = version.toLowerCase();
  return !unstableKeywords.some((keyword) => lower.includes(keyword));
}

async function parseCurrentVersions() {
  const content = await Bun.file(BUILD_GRADLE).text();
  const deps: Record<string, string> = {};

  const regex = /implementation\("([^:]+):([^:]+):([^"]+)"\)/g;
  let match: RegExpExecArray | null = regex.exec(content);

  while (match !== null) {
    const [, group, artifact, version] = match;
    if (group && artifact && version) {
      deps[`${group}:${artifact}`] = version;
    }
    match = regex.exec(content);
  }

  return deps;
}

async function checkMavenVersion(group: string, artifact: string) {
  try {
    const url = `https://search.maven.org/solrsearch/select?q=g:${encodeURIComponent(group)}+AND+a:${encodeURIComponent(artifact)}&rows=50&wt=json&core=gav`;
    const response = await fetch(url);
    const data = (await response.json()) as {
      response: { docs: Array<{ v: string }> };
    };

    const versions = [...new Set(data.response.docs.map((doc) => doc.v))];
    const stableVersions = versions.filter(isStableVersion);

    return stableVersions[0] || versions[0] || null;
  } catch (_error) {
    return null;
  }
}

async function checkGoogleMavenVersion(group: string, artifact: string) {
  try {
    const groupPath = group.replace(/\./g, '/');
    const url = `https://maven.google.com/${groupPath}/${artifact}/maven-metadata.xml`;
    const response = await fetch(url);
    const xml = await response.text();

    const versionMatches = xml.matchAll(/<version>(.*?)<\/version>/g);
    const versions = Array.from(versionMatches, (m) => m[1]).filter((v): v is string => !!v);

    const stableVersions = versions.filter(isStableVersion);
    return stableVersions[stableVersions.length - 1] ?? versions[versions.length - 1] ?? null;
  } catch (_error) {
    return null;
  }
}

async function checkUnifiedPushVersion() {
  try {
    const response = await fetch('https://api.github.com/repos/UnifiedPush/android-connector/tags');
    const tags = (await response.json()) as Array<{ name: string }>;
    return tags[0]?.name || null;
  } catch (_error) {
    return null;
  }
}

async function main() {
  console.log('Checking for Android dependency updates...\n');

  const dependencies = await parseCurrentVersions();
  let hasUpdates = false;

  for (const [dep, currentVersion] of Object.entries(dependencies)) {
    const [group, artifact] = dep.split(':');
    if (!group || !artifact) continue;
    let latestVersion: string | null = null;

    if (dep === 'com.github.UnifiedPush:android-connector') {
      latestVersion = await checkUnifiedPushVersion();
    } else if (group.startsWith('androidx') || group === 'com.google.android.material') {
      latestVersion = await checkGoogleMavenVersion(group, artifact);
    } else {
      latestVersion = await checkMavenVersion(group, artifact);
    }

    if (!latestVersion) {
      console.log(`❌ ${dep}: Could not check`);
      continue;
    }

    if (latestVersion === currentVersion) {
      console.log(`✓ ${dep}: ${currentVersion} (latest)`);
    } else {
      console.log(`⚠ ${dep}: ${currentVersion} → ${latestVersion}`);
      hasUpdates = true;
    }
  }

  if (hasUpdates) {
    console.log('\nUpdates available! Edit android/app/build.gradle.kts to upgrade.');
  } else {
    console.log('\nAll dependencies are up to date.');
  }
}

main().catch((error) => {
  console.error('Failed to check updates:', error.message);
  process.exit(1);
});
