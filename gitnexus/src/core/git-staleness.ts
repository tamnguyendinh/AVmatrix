/**
 * Git working tree vs index commit staleness (used by MCP resources, group status, etc.).
 * Lives in core/ so application code does not depend on the MCP package layer.
 */

import { execFileSync } from 'node:child_process';

export interface StalenessInfo {
  isStale: boolean;
  commitsBehind: number;
  hint?: string;
}

/**
 * Check how many commits the index is behind HEAD (synchronous; uses git CLI).
 */
export function checkStaleness(repoPath: string, lastCommit: string): StalenessInfo {
  try {
    const result = execFileSync('git', ['rev-list', '--count', `${lastCommit}..HEAD`], {
      cwd: repoPath,
      encoding: 'utf-8',
      stdio: ['pipe', 'pipe', 'pipe'],
    }).trim();

    const commitsBehind = parseInt(result, 10) || 0;

    if (commitsBehind > 0) {
      return {
        isStale: true,
        commitsBehind,
        hint: `⚠️ Index is ${commitsBehind} commit${commitsBehind > 1 ? 's' : ''} behind HEAD. Run analyze tool to update.`,
      };
    }

    return { isStale: false, commitsBehind: 0 };
  } catch {
    return { isStale: false, commitsBehind: 0 };
  }
}
