import { describe, it, expect } from 'vitest';
import type { GraphNode, GraphRelationship } from 'avmatrix-shared';
import { createKnowledgeGraph } from '../../src/core/graph/graph.js';
import {
  getCommunityColor,
  COMMUNITY_COLORS,
  processCommunities,
} from '../../src/core/ingestion/community-processor.js';

describe('community-processor', () => {
  describe('COMMUNITY_COLORS', () => {
    it('has 12 colors', () => {
      expect(COMMUNITY_COLORS).toHaveLength(12);
    });

    it('contains valid hex color strings', () => {
      for (const color of COMMUNITY_COLORS) {
        expect(color).toMatch(/^#[0-9a-fA-F]{6}$/);
      }
    });

    it('has no duplicate colors', () => {
      const unique = new Set(COMMUNITY_COLORS);
      expect(unique.size).toBe(COMMUNITY_COLORS.length);
    });
  });

  describe('getCommunityColor', () => {
    it('returns first color for index 0', () => {
      expect(getCommunityColor(0)).toBe(COMMUNITY_COLORS[0]);
    });

    it('wraps around when index exceeds color count', () => {
      expect(getCommunityColor(12)).toBe(COMMUNITY_COLORS[0]);
      expect(getCommunityColor(13)).toBe(COMMUNITY_COLORS[1]);
    });

    it('returns different colors for different indices', () => {
      const c0 = getCommunityColor(0);
      const c1 = getCommunityColor(1);
      expect(c0).not.toBe(c1);
    });
  });

  describe('processCommunities', () => {
    it('returns deterministic community output for the same graph input', async () => {
      const first = await processCommunities(createCommunityFixture());
      const second = await processCommunities(createCommunityFixture());

      expect(first).toEqual(second);
    });
  });
});

function createCommunityFixture(): ReturnType<typeof createKnowledgeGraph> {
  const graph = createKnowledgeGraph();
  addClique(graph, 'auth', 'auth', 5);
  addClique(graph, 'billing', 'billing', 5);
  graph.addRelationship(makeRel('bridge:auth-billing', 'fn:auth0', 'fn:billing0', 0.2));
  return graph;
}

function addClique(
  graph: ReturnType<typeof createKnowledgeGraph>,
  prefix: string,
  folder: string,
  size: number,
): void {
  const ids: string[] = [];
  for (let i = 0; i < size; i++) {
    const id = `fn:${prefix}${i}`;
    ids.push(id);
    graph.addNode(makeNode(id, `${prefix}Fn${i}`, `/src/${folder}/file${i}.ts`));
  }

  let relIndex = 0;
  for (let i = 0; i < ids.length; i++) {
    for (let j = i + 1; j < ids.length; j++) {
      graph.addRelationship(makeRel(`rel:${prefix}:${relIndex++}`, ids[i], ids[j], 1));
    }
  }
}

function makeNode(id: string, name: string, filePath: string): GraphNode {
  return {
    id,
    label: 'Function',
    properties: { name, filePath, startLine: 1, endLine: 1 },
  };
}

function makeRel(
  id: string,
  sourceId: string,
  targetId: string,
  confidence: number,
): GraphRelationship {
  return { id, type: 'CALLS', sourceId, targetId, confidence, reason: 'test' };
}
