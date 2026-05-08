import { describe, expect, it } from 'vitest';
import type { BackendRepo } from '../../src/services/backend-client';
import { includeRepoInList } from '../../src/services/repo-list';

const repo = (
  overrides: Partial<BackendRepo> & Pick<BackendRepo, 'name' | 'path'>,
): BackendRepo => ({
  indexedAt: '2026-05-08T00:00:00.000Z',
  ...overrides,
});

describe('repo list merge', () => {
  it('adds the analyzed repo when the refreshed backend list is still stale', () => {
    const existing = repo({ name: 'AVmatrix', path: 'F:\\AVmatrix-main' });
    const analyzed = repo({
      name: 'restaurant_manager',
      path: 'E:\\Lap_trinh\\restaurant_manager',
      stats: { nodes: 1200, edges: 3400 },
    });

    expect(includeRepoInList([existing], analyzed)).toEqual([analyzed, existing]);
  });

  it('updates an existing dropdown entry using the canonical repo path', () => {
    const stale = repo({
      name: 'restaurant_manager',
      path: 'E:/Lap_trinh/restaurant_manager',
      indexedAt: '2026-05-07T00:00:00.000Z',
      stats: { nodes: 10, edges: 12 },
    });
    const fresh = repo({
      name: 'restaurant_manager',
      path: 'E:\\Lap_trinh\\restaurant_manager\\',
      indexedAt: '2026-05-08T00:00:00.000Z',
      stats: { nodes: 1200, edges: 3400 },
    });

    expect(includeRepoInList([stale], fresh)).toEqual([fresh]);
  });

  it('does not replace a different path that shares the same repo name', () => {
    const first = repo({
      name: 'demo',
      path: 'F:\\one\\demo',
      stats: { nodes: 10 },
    });
    const second = repo({
      name: 'demo',
      path: 'F:\\two\\demo',
      stats: { nodes: 20 },
    });

    expect(includeRepoInList([first], second)).toEqual([second, first]);
  });
});
