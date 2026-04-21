import { describe, expect, it } from 'vitest';
import { HOLD_QUEUE_TIMEOUT_SECS } from '../../src/server/api.js';

describe('repo hold-queue timeout contract', () => {
  it('waits up to 10 minutes for large local repo analysis to finish', () => {
    expect(HOLD_QUEUE_TIMEOUT_SECS).toBe(600);
  });
});
