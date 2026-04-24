import { describe, expect, it } from 'vitest';
import { createKnowledgeGraph } from '../../src/core/graph/graph.js';
import {
  compareGraphCorrectnessSnapshots,
  createGraphCorrectnessSnapshot,
} from '../../src/core/analyze/graph-correctness-snapshot.js';

describe('graph correctness snapshot', () => {
  it('detects node property and relationship changes', () => {
    const before = createKnowledgeGraph();
    before.addNode({
      id: 'File:src/a.ts',
      label: 'File',
      properties: { name: 'a.ts', filePath: 'src/a.ts' },
    });
    before.addNode({
      id: 'Function:src/a.ts:run',
      label: 'Function',
      properties: { name: 'run', filePath: 'src/a.ts', startLine: 1, endLine: 3 },
    });
    before.addRelationship({
      id: 'rel-1',
      type: 'CONTAINS',
      sourceId: 'File:src/a.ts',
      targetId: 'Function:src/a.ts:run',
      confidence: 1,
      reason: 'test',
    });

    const after = createKnowledgeGraph();
    after.addNode({
      id: 'File:src/a.ts',
      label: 'File',
      properties: { name: 'a.ts', filePath: 'src/a.ts' },
    });
    after.addNode({
      id: 'Function:src/a.ts:run',
      label: 'Function',
      properties: { name: 'runFast', filePath: 'src/a.ts', startLine: 1, endLine: 3 },
    });
    after.addRelationship({
      id: 'rel-1',
      type: 'CALLS',
      sourceId: 'File:src/a.ts',
      targetId: 'Function:src/a.ts:run',
      confidence: 1,
      reason: 'test',
    });

    const diffs = compareGraphCorrectnessSnapshots(
      createGraphCorrectnessSnapshot(before),
      createGraphCorrectnessSnapshot(after),
    );

    expect(diffs.map((d) => d.field)).toEqual(
      expect.arrayContaining(['byRelationshipType', 'nodeDigest', 'relationshipDigest']),
    );
  });

  it('treats equivalent graphs as equal regardless of insertion order', () => {
    const first = createKnowledgeGraph();
    const second = createKnowledgeGraph();
    const nodes = [
      {
        id: 'File:src/a.ts',
        label: 'File' as const,
        properties: { name: 'a.ts', filePath: 'src/a.ts' },
      },
      {
        id: 'Function:src/a.ts:run',
        label: 'Function' as const,
        properties: { name: 'run', filePath: 'src/a.ts' },
      },
    ];
    const rel = {
      id: 'rel-1',
      type: 'CONTAINS' as const,
      sourceId: 'File:src/a.ts',
      targetId: 'Function:src/a.ts:run',
      confidence: 1,
      reason: 'test',
    };

    first.addNode(nodes[0]);
    first.addNode(nodes[1]);
    first.addRelationship(rel);
    second.addNode(nodes[1]);
    second.addNode(nodes[0]);
    second.addRelationship(rel);

    expect(
      compareGraphCorrectnessSnapshots(
        createGraphCorrectnessSnapshot(first),
        createGraphCorrectnessSnapshot(second),
      ),
    ).toEqual([]);
  });
});
