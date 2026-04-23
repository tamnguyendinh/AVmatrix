import { describe, expect, it } from 'vitest';
import {
  IMPACT_DEFAULTS,
  IMPACT_DEFAULT_RELATION_TYPES,
  parseImpactInput,
} from '../../src/mcp/contracts/impact.js';

describe('impact contract parser', () => {
  it('accepts target_uid-only requests and applies runtime defaults', () => {
    const result = parseImpactInput({
      target_uid: 'uid:1234',
      direction: 'upstream',
    });

    expect(result.ok).toBe(true);
    if (!result.ok) return;

    expect(result.value.target).toBeUndefined();
    expect(result.value.target_uid).toBe('uid:1234');
    expect(result.value.direction).toBe('upstream');
    expect(result.value.maxDepth).toBe(IMPACT_DEFAULTS.maxDepth);
    expect(result.value.includeTests).toBe(IMPACT_DEFAULTS.includeTests);
    expect(result.value.minConfidence).toBe(IMPACT_DEFAULTS.minConfidence);
    expect(result.value.relationTypes).toEqual([...IMPACT_DEFAULT_RELATION_TYPES]);
  });

  it('rejects requests missing both target and target_uid', () => {
    const result = parseImpactInput({
      direction: 'upstream',
    });

    expect(result.ok).toBe(false);
    if (result.ok) return;

    expect(result.error.field).toBe('target');
  });

  it('rejects invalid direction instead of silently coercing it', () => {
    const result = parseImpactInput({
      target: 'AuthService',
      direction: 'upstrem',
    });

    expect(result.ok).toBe(false);
    if (result.ok) return;

    expect(result.error.field).toBe('direction');
    expect(result.error.allowedValues).toEqual(['upstream', 'downstream']);
  });

  it('rejects invalid relationTypes instead of falling back to defaults', () => {
    const result = parseImpactInput({
      target: 'AuthService',
      direction: 'upstream',
      relationTypes: ['NOT_A_RELATION'],
    });

    expect(result.ok).toBe(false);
    if (result.ok) return;

    expect(result.error.field).toBe('relationTypes');
  });

  it('preserves legacy OVERRIDES alias while expanding METHOD_OVERRIDES for traversal', () => {
    const result = parseImpactInput({
      target: 'AuthService',
      direction: 'upstream',
      relationTypes: ['OVERRIDES'],
    });

    expect(result.ok).toBe(true);
    if (!result.ok) return;

    expect(result.value.relationTypes).toEqual(['OVERRIDES', 'METHOD_OVERRIDES']);
  });
});
