import type { BackendRepo } from './backend-client';

const normalizeRepoIdentityPath = (repo: BackendRepo): string =>
  (repo.repoPath ?? repo.path).replace(/\\/g, '/').replace(/\/+$/, '');

export const includeRepoInList = (repos: BackendRepo[], repo: BackendRepo): BackendRepo[] => {
  const repoPath = normalizeRepoIdentityPath(repo);
  const existingIndex = repos.findIndex((item) => {
    const itemPath = normalizeRepoIdentityPath(item);
    return repoPath.length > 0 ? itemPath === repoPath : item.name === repo.name;
  });

  if (existingIndex === -1) return [repo, ...repos];

  const nextRepos = [...repos];
  nextRepos[existingIndex] = { ...nextRepos[existingIndex], ...repo };
  return nextRepos;
};
