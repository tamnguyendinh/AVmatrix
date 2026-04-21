import os from 'os';
import path from 'path';

/**
 * Return the legacy GitNexus cache directory that older remote-clone flows used.
 *
 * Local-only mode no longer creates or updates this cache, but delete flows
 * still clean it up if a user has leftovers from older versions.
 */
export function getLegacyRepoCacheDir(repoName: string): string {
  return path.join(os.homedir(), '.gitnexus', 'repos', repoName);
}
