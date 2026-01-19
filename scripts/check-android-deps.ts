import { join } from 'node:path';

const BUILD_GRADLE = join(import.meta.dir, '..', 'android', 'app', 'build.gradle.kts');

function compareVersions(a: string, b: string): number {
  const parseVersion = (v: string) => {
    const parts = v.split(/[.-]/).map((p) => {
      const num = Number.parseInt(p, 10);
      return Number.isNaN(num) ? p : num;
    });
    return parts;
  };

  const aParts = parseVersion(a);
  const bParts = parseVersion(b);
  const maxLen = Math.max(aParts.length, bParts.length);

  for (let i = 0; i < maxLen; i++) {
    const aPart = aParts[i] ?? 0;
    const bPart = bParts[i] ?? 0;

    if (typeof aPart === 'number' && typeof bPart === 'number') {
      if (aPart !== bPart) return aPart - bPart;
    } else {
      const aStr = String(aPart);
      const bStr = String(bPart);
      if (aStr !== bStr) return aStr < bStr ? -1 : 1;
    }
  }
  return 0;
}

function isStableVersion(version: string) {
  const unstableKeywords = ['alpha', 'beta', 'rc', 'snapshot', 'dev', 'preview'];
  const lower = version.toLowerCase();
  return !unstableKeywords.some((keyword) => lower.includes(keyword));
}

async function parseCurrentVersions() {
  const content = await Bun.file(BUILD_GRADLE).text();
  const deps: Record<string, string> = {};
  const variables: Record<string, string> = {};

  const varRegex = /val\s+(\w+)\s*=\s*"([^"]+)"/g;
  let varMatch: RegExpExecArray | null = varRegex.exec(content);
  while (varMatch !== null) {
    const [, varName, varValue] = varMatch;
    if (varName && varValue) {
      variables[varName] = varValue;
    }
    varMatch = varRegex.exec(content);
  }

  const depRegex = /implementation\("([^:]+):([^:]+):([^"]+)"\)/g;
  let depMatch: RegExpExecArray | null = depRegex.exec(content);

  while (depMatch !== null) {
    const [, group, artifact, version] = depMatch;
    if (group && artifact && version) {
      const varMatch = version.match(/\$(\w+)/);
      const resolvedVersion = varMatch?.[1] ? (variables[varMatch[1]] ?? version) : version;
      deps[`${group}:${artifact}`] = resolvedVersion;
    }
    depMatch = depRegex.exec(content);
  }

  return deps;
}

async function checkMavenVersion(group: string, artifact: string) {
  try {
    const url = `https://search.maven.org/solrsearch/select?q=g:${encodeURIComponent(group)}+AND+a:${encodeURIComponent(artifact)}&rows=100&wt=json&core=gav`;
    const response = await fetch(url);
    const data = (await response.json()) as {
      response: { docs: Array<{ v: string }> };
    };

    const versions = [...new Set(data.response.docs.map((doc) => doc.v))];
    const stableVersions = versions.filter(isStableVersion);
    const sortedVersions = stableVersions.sort(compareVersions);

    return (
      sortedVersions[sortedVersions.length - 1] ||
      versions.sort(compareVersions)[versions.length - 1] ||
      null
    );
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
    const sortedVersions = stableVersions.sort(compareVersions);
    return (
      sortedVersions[sortedVersions.length - 1] ??
      versions.sort(compareVersions)[versions.length - 1] ??
      null
    );
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
      console.log(`${dep}: Could not check`);
      continue;
    }

    if (latestVersion === currentVersion) {
      console.log(`${dep}: ${currentVersion} (latest)`);
    } else if (compareVersions(currentVersion, latestVersion) < 0) {
      console.log(` ${dep}: ${currentVersion} → ${latestVersion}`);
      hasUpdates = true;
    } else {
      console.log(`ℹ ${dep}: ${currentVersion} (newer than Maven: ${latestVersion})`);
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
