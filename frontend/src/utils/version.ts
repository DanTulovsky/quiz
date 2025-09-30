// Version utility for automatic versioning
export interface AppVersion {
  version: string;
  buildTime: string;
  commitHash: string;
}

// Get version from environment or generate one
export function getAppVersion(): AppVersion {
  const buildTime = import.meta.env.VITE_BUILD_TIME || new Date().toISOString();
  const commitHash = import.meta.env.VITE_COMMIT_HASH || 'dev';
  const version = import.meta.env.VITE_APP_VERSION || '1.0.0';

  return {
    version,
    buildTime,
    commitHash: commitHash.substring(0, 8), // Short hash
  };
}

// Format version for display
export function formatVersion(version: AppVersion): string {
  const date = new Date(version.buildTime).toLocaleDateString();
  return `${version.version} (${date})`;
}
