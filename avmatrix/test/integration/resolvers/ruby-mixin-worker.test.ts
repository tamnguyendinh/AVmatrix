/**
 * Regression: Ruby mixin heritage resolution must work on the canonical worker
 * ingestion path.
 *
 * This file used to compare the removed legacy parser and worker output. Full analyze
 * now treats worker parsing as canonical, so the guard focuses on worker output
 * and verifies that the old sequential path is not required for correctness.
 */
import { describe, it, expect, beforeAll } from 'vitest';
import path from 'path';
import type { GraphRelationship } from '../../../src/core/graph/types.js';
import {
  FIXTURES,
  getRelationships,
  getNodesByLabel,
  runPipelineFromRepo,
  type PipelineResult,
} from './helpers.js';

const FIXTURE = path.join(FIXTURES, 'ruby-sequential-mixin');

/** CALLS edges from `sourceName` whose target is a Method node. */
function methodCallEdges(result: PipelineResult, sourceName: string): Set<string> {
  const edges = getRelationships(result, 'CALLS').filter(
    (e) => e.source === sourceName && e.targetLabel === 'Method',
  );
  return new Set(edges.map((e) => `${e.source} → ${e.target}`));
}

function findMethodOwner(result: PipelineResult, methodNodeId: string): string | undefined {
  for (const rel of result.graph.iterRelationships() as IterableIterator<GraphRelationship>) {
    if (rel.type === 'HAS_METHOD' && rel.targetId === methodNodeId) {
      return result.graph.getNode(rel.sourceId)?.properties.name;
    }
  }
  return undefined;
}

function resolvedMethodOwners(
  result: PipelineResult,
  sourceName: string,
  targetMethodName: string,
): string[] {
  const owners: string[] = [];
  for (const e of getRelationships(result, 'CALLS')) {
    if (e.source === sourceName && e.targetLabel === 'Method' && e.target === targetMethodName) {
      const owner = findMethodOwner(result, e.rel.targetId);
      if (owner) owners.push(owner);
    }
  }
  return owners.sort();
}

describe('Ruby mixin heritage via worker path', () => {
  let result: PipelineResult;

  beforeAll(async () => {
    result = await runPipelineFromRepo(FIXTURE, () => {});
  }, 120000);

  it('uses the worker path', () => {
    expect(result.usedWorkerPool).toBe(true);
  });

  it('labels Ruby modules as Trait', () => {
    const expected = ['Greetable', 'LoggerMixin', 'PrependedOverride'];
    expect(getNodesByLabel(result, 'Trait').sort()).toEqual(expected);
    expect(getNodesByLabel(result, 'Module').filter((n) => n !== 'lib')).toEqual([]);
  });

  it('resolves include-provided method: call_greet → Greetable#greet', () => {
    const edges = methodCallEdges(result, 'call_greet');
    expect([...edges]).toContain('call_greet → greet');
    expect(resolvedMethodOwners(result, 'call_greet', 'greet')).toContain('Greetable');
  });

  it('resolves prepend-only method: call_prepended_marker → PrependedOverride#prepended_marker', () => {
    expect(resolvedMethodOwners(result, 'call_prepended_marker', 'prepended_marker')).toContain(
      'PrependedOverride',
    );
  });

  it('emits IMPLEMENTS edges for all three mixin kinds', () => {
    const kinds = getRelationships(result, 'IMPLEMENTS')
      .filter((e) => e.source === 'Account')
      .map((e) => e.rel.reason ?? '')
      .sort();
    expect(kinds).toEqual(['extend', 'include', 'prepend']);
  });
});
