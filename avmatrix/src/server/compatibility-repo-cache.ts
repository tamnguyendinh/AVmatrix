import path from 'path';
import { getGlobalDir } from '../storage/repo-manager.js';

/**
 * Return the compatibility AVmatrix cache directory that older remote-clone flows used.
 *
 * Local-only mode no longer creates or updates this cache, but delete flows
 * still clean it up if a user has leftovers from older versions.
 */
export function getCompatibilityRepoCacheDir(repoName: string): string {
  return path.join(getGlobalDir(), 'repos', repoName);
}
