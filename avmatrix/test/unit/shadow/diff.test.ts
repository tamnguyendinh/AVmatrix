/**
 * Unit tests for `diffResolutions` for the accurate single-pass graph pipeline.
 *
 * Pins the 5 `ShadowAgreement` outcomes and the symmetric-by-kind evidence-
 * delta contract. Inputs are pure data fixtures — no real pipeline state.
 */

import { describe, it, expect } from 'vitest';
import {
  diffResolutions,
  type Resolution,
  type ResolutionEvidence,
  type ShadowCallsite,
  type SymbolDefinition,
} from 'avmatrix-shared';

// ─── Fixtures ───────────────────────────────────────────────────────────────

const callsite: ShadowCallsite = {
  filePath: 'src/app.ts',
  line: 42,
  col: 8,
  calledName: 'save',
};

const makeDef = (nodeId: string): SymbolDefinition => ({
  nodeId,
  filePath: 'src/models.ts',
  type: 'Method',
});

const makeEvidence = (kind: ResolutionEvidence['kind'], weight = 0.5): ResolutionEvidence => ({
  kind,
  weight,
});

const makeResolution = (
  nodeId: string,
  evidenceKinds: readonly ResolutionEvidence['kind'][],
): Resolution => ({
  def: makeDef(nodeId),
  confidence: Math.min(1, evidenceKinds.length * 0.3),
  evidence: evidenceKinds.map((k) => makeEvidence(k)),
});

// ─── Agreement outcomes ─────────────────────────────────────────────────────

describe('diffResolutions — agreement outcomes', () => {
  it("both arrays empty → 'both-empty' with no evidence delta", () => {
    const result = diffResolutions(callsite, [], []);
    expect(result.agreement).toBe('both-empty');
    expect(result.evidenceDelta).toEqual([]);
    expect(result.baseline).toBeNull();
    expect(result.newResult).toBeNull();
  });

  it("identical top DefIds → 'both-agree' with empty evidence delta", () => {
    const baseline = [makeResolution('def:User.save', ['local', 'owner-match'])];
    const next = [makeResolution('def:User.save', ['local', 'kind-match'])];
    const result = diffResolutions(callsite, baseline, next);
    expect(result.agreement).toBe('both-agree');
    expect(result.evidenceDelta).toEqual([]);
    expect(result.baseline).toBe(baseline[0]);
    expect(result.newResult).toBe(next[0]);
  });

  it("baseline empty, new non-empty → 'only-new' with new's evidence as delta", () => {
    const next = [makeResolution('def:User.save', ['local', 'owner-match'])];
    const result = diffResolutions(callsite, [], next);
    expect(result.agreement).toBe('only-new');
    expect(result.evidenceDelta).toEqual(next[0].evidence);
    expect(result.baseline).toBeNull();
    expect(result.newResult).toBe(next[0]);
  });

  it("baseline non-empty, new empty → 'only-baseline' with baseline's evidence as delta", () => {
    const baseline = [makeResolution('def:User.save', ['global-name'])];
    const result = diffResolutions(callsite, baseline, []);
    expect(result.agreement).toBe('only-baseline');
    expect(result.evidenceDelta).toEqual(baseline[0].evidence);
    expect(result.baseline).toBe(baseline[0]);
    expect(result.newResult).toBeNull();
  });

  it("different top DefIds → 'both-disagree'", () => {
    const baseline = [makeResolution('def:ModelA.save', ['global-name'])];
    const next = [makeResolution('def:ModelB.save', ['local'])];
    const result = diffResolutions(callsite, baseline, next);
    expect(result.agreement).toBe('both-disagree');
    expect(result.baseline).toBe(baseline[0]);
    expect(result.newResult).toBe(next[0]);
  });
});

// ─── Evidence delta — symmetric difference by `kind` ────────────────────────

describe('diffResolutions — evidence delta (symmetric-by-kind)', () => {
  it("'both-disagree' with disjoint evidence → delta contains both sides' kinds", () => {
    const baseline = [makeResolution('def:A', ['global-name'])];
    const next = [makeResolution('def:B', ['local', 'owner-match'])];
    const result = diffResolutions(callsite, baseline, next);
    expect(result.evidenceDelta.map((e) => e.kind)).toEqual([
      'global-name',
      'local',
      'owner-match',
    ]);
  });

  it("'both-disagree' with overlapping kinds → overlapping kinds removed from delta", () => {
    const baseline = [makeResolution('def:A', ['local', 'scope-chain', 'global-name'])];
    const next = [makeResolution('def:B', ['local', 'import', 'owner-match'])];
    const result = diffResolutions(callsite, baseline, next);
    // 'local' is on both sides → dropped
    // Remaining: baseline-only ['scope-chain', 'global-name'], then new-only ['import', 'owner-match']
    expect(result.evidenceDelta.map((e) => e.kind)).toEqual([
      'scope-chain',
      'global-name',
      'import',
      'owner-match',
    ]);
  });

  it("'both-disagree' with fully overlapping kinds → empty evidence delta", () => {
    const baseline = [makeResolution('def:A', ['local', 'owner-match'])];
    const next = [makeResolution('def:B', ['owner-match', 'local'])];
    const result = diffResolutions(callsite, baseline, next);
    // Same kind set, different order → symmetric difference is empty
    expect(result.evidenceDelta).toEqual([]);
    expect(result.agreement).toBe('both-disagree'); // agreement still disagrees because nodeIds differ
  });

  it('differing weights on the same kind → NOT a delta (keyed on kind only)', () => {
    const baseline = [
      {
        def: makeDef('def:A'),
        confidence: 0.9,
        evidence: [{ kind: 'local' as const, weight: 0.55 }],
      },
    ];
    const next = [
      {
        def: makeDef('def:B'),
        confidence: 0.1,
        evidence: [{ kind: 'local' as const, weight: 0.25 }],
      },
    ];
    const result = diffResolutions(callsite, baseline, next);
    expect(result.agreement).toBe('both-disagree');
    expect(result.evidenceDelta).toEqual([]);
  });
});

// ─── Metadata + ordering ────────────────────────────────────────────────────

describe('diffResolutions — metadata + ordering', () => {
  it('ignores resolutions beyond index 0 (top match only)', () => {
    const baseline = [
      makeResolution('def:User.save', ['local']),
      makeResolution('def:other', ['global-name']),
    ];
    const next = [
      makeResolution('def:User.save', ['local']),
      // The 2nd entry is here to verify index-0 isolation — the only kind
      // requirement is that it be a valid `ResolutionEvidence.kind` so the
      // fixture is type-correct. `'global-name'` is a real kind that
      // `diffResolutions` never treats specially.
      makeResolution('def:yet-another', ['global-name']),
    ];
    const result = diffResolutions(callsite, baseline, next);
    expect(result.agreement).toBe('both-agree');
  });

  it('preserves callsite verbatim', () => {
    const result = diffResolutions(callsite, [], []);
    expect(result.callsite).toBe(callsite);
  });

  it("'both-disagree' delta order: baseline-only first (input order), then new-only", () => {
    const baseline = [makeResolution('def:A', ['owner-match', 'scope-chain', 'kind-match'])];
    const next = [makeResolution('def:B', ['import', 'owner-match', 'arity-match'])];
    const result = diffResolutions(callsite, baseline, next);
    // 'owner-match' overlaps → dropped
    // baseline-only in original order: ['scope-chain', 'kind-match']
    // then new-only in original order: ['import', 'arity-match']
    expect(result.evidenceDelta.map((e) => e.kind)).toEqual([
      'scope-chain',
      'kind-match',
      'import',
      'arity-match',
    ]);
  });
});
