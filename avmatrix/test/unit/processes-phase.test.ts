import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  calculateDynamicMaxProcesses,
  DEFAULT_MAX_PROCESSES_CAP,
  resolveConfiguredMaxProcessesCap,
} from '../../src/core/ingestion/pipeline-phases/processes.js';

describe('processes phase maxProcesses config', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('defaults to 700 when env and saved config are unset', async () => {
    const cap = await resolveConfiguredMaxProcessesCap(undefined, async () => ({}));
    expect(cap).toBe(DEFAULT_MAX_PROCESSES_CAP);
  });

  it('prefers AVMATRIX_MAX_PROCESSES over saved config', async () => {
    vi.stubEnv('AVMATRIX_MAX_PROCESSES', '650');

    const cap = await resolveConfiguredMaxProcessesCap(process.env.AVMATRIX_MAX_PROCESSES, async () => ({
      maxProcesses: 900,
    }));

    expect(cap).toBe(650);
  });

  it('uses saved config when env is unset', async () => {
    const cap = await resolveConfiguredMaxProcessesCap(undefined, async () => ({
      maxProcesses: 550,
    }));

    expect(cap).toBe(550);
  });

  it('falls back to default when configured values are invalid', async () => {
    vi.stubEnv('AVMATRIX_MAX_PROCESSES', 'abc');

    const cap = await resolveConfiguredMaxProcessesCap(process.env.AVMATRIX_MAX_PROCESSES, async () => ({
      maxProcesses: -10,
    }));

    expect(cap).toBe(DEFAULT_MAX_PROCESSES_CAP);
  });

  it('applies the configured cap after the dynamic scaling floor', () => {
    expect(calculateDynamicMaxProcesses(16059, 700)).toBe(700);
    expect(calculateDynamicMaxProcesses(16059, 300)).toBe(300);
    expect(calculateDynamicMaxProcesses(15, 700)).toBe(20);
    expect(calculateDynamicMaxProcesses(15, 10)).toBe(10);
  });
});
