import {
  executeQuery as executePoolQuery,
  executeParameterized as executePoolParameterized,
  initLbug,
  streamQuery as streamPoolQuery,
  touchRepo,
} from '../../core/lbug/pool-adapter.js';

export interface RepoReadTarget {
  repoId: string;
  lbugPath: string;
}

export const ensureRepoReadReady = async (target: RepoReadTarget): Promise<void> => {
  await initLbug(target.repoId, target.lbugPath);
  touchRepo(target.repoId);
};

export const executeRepoReadQuery = async (
  target: RepoReadTarget,
  cypher: string,
): Promise<any[]> => {
  await ensureRepoReadReady(target);
  return executePoolQuery(target.repoId, cypher);
};

export const executeRepoParameterizedReadQuery = async (
  target: RepoReadTarget,
  cypher: string,
  params: Record<string, any>,
): Promise<any[]> => {
  await ensureRepoReadReady(target);
  return executePoolParameterized(target.repoId, cypher, params);
};

export const streamRepoReadQuery = async (
  target: RepoReadTarget,
  cypher: string,
  onRow: (row: any) => void | Promise<void>,
): Promise<number> => {
  await ensureRepoReadReady(target);
  return streamPoolQuery(target.repoId, cypher, onRow);
};
