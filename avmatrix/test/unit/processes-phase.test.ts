import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  calculateDynamicMaxProcesses,
  DEFAULT_MAX_PROCESSES_CAP,
  resolveConfiguredMaxProcessesCap,
} from '../../src/core/ingestion/pipeline-phases/processes.js';
import type { AVmatrixSettings } from '../../src/storage/settings.js';

describe('processes phase maxProcesses config', () => {
  const repoPath = '/tmp/demo-repo';

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('defaults to 700 when env and settings are unset', async () => {
    const cap = await resolveConfiguredMaxProcessesCap(repoPath, undefined, async () => ({
      maxExecutionFlows: DEFAULT_MAX_PROCESSES_CAP,
    }));
    expect(cap).toBe(DEFAULT_MAX_PROCESSES_CAP);
  });

  it('prefers AVMATRIX_MAX_PROCESSES over settings.json', async () => {
    vi.stubEnv('AVMATRIX_MAX_PROCESSES', '650');

    const cap = await resolveConfiguredMaxProcessesCap(
      repoPath,
      process.env.AVMATRIX_MAX_PROCESSES,
      async (_repoPath: string): Promise<AVmatrixSettings> => ({
        maxExecutionFlows: 900,
      }),
    );

    expect(cap).toBe(650);
  });

  it('uses settings.json when env is unset', async () => {
    const cap = await resolveConfiguredMaxProcessesCap(
      repoPath,
      undefined,
      async (_repoPath: string): Promise<AVmatrixSettings> => ({
        maxExecutionFlows: 550,
      }),
    );

    expect(cap).toBe(550);
  });

  it('falls back to default when configured values are invalid', async () => {
    vi.stubEnv('AVMATRIX_MAX_PROCESSES', 'abc');

    const cap = await resolveConfiguredMaxProcessesCap(
      repoPath,
      process.env.AVMATRIX_MAX_PROCESSES,
      async (_repoPath: string): Promise<AVmatrixSettings> => ({
        maxExecutionFlows: -10 as never,
      }),
    );

    expect(cap).toBe(DEFAULT_MAX_PROCESSES_CAP);
  });

  it('applies the configured cap after the dynamic scaling floor', () => {
    expect(calculateDynamicMaxProcesses(16059, 700)).toBe(700);
    expect(calculateDynamicMaxProcesses(16059, 300)).toBe(300);
    expect(calculateDynamicMaxProcesses(15, 700)).toBe(20);
    expect(calculateDynamicMaxProcesses(15, 10)).toBe(10);
  });
});
