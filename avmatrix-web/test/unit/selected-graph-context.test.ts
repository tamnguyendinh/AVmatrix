import { describe, expect, it } from 'vitest';
import Graph from 'graphology';
import { buildSelectedGraphContext } from '../../src/lib/selected-graph-context';
import type { SigmaNodeAttributes, SigmaEdgeAttributes } from '../../src/lib/graph-adapter';

const createGraph = () => {
  const graph = new Graph<SigmaNodeAttributes, SigmaEdgeAttributes>();
  graph.addNode('selected', {
    x: 0,
    y: 0,
    size: 4,
    color: '#fff',
    label: 'selected',
    nodeType: 'File',
    filePath: 'src/selected.ts',
  });
  graph.addNode('neighbor-a', {
    x: 1,
    y: 1,
    size: 4,
    color: '#fff',
    label: 'neighbor-a',
    nodeType: 'Function',
    filePath: 'src/a.ts',
  });
  graph.addNode('neighbor-b', {
    x: 2,
    y: 2,
    size: 4,
    color: '#fff',
    label: 'neighbor-b',
    nodeType: 'Function',
    filePath: 'src/b.ts',
  });
  graph.addNode('isolated', {
    x: 3,
    y: 3,
    size: 4,
    color: '#fff',
    label: 'isolated',
    nodeType: 'Function',
    filePath: 'src/isolated.ts',
  });

  graph.addEdgeWithKey('edge-out', 'selected', 'neighbor-a', {
    size: 1,
    color: '#000',
    relationType: 'CALLS',
  });
  graph.addEdgeWithKey('edge-in', 'neighbor-b', 'selected', {
    size: 1,
    color: '#000',
    relationType: 'IMPORTS',
  });

  return graph;
};

describe('selected graph context', () => {
  it('returns direct neighbor node ids and direct edge ids for the selected node', () => {
    const graph = createGraph();

    const context = buildSelectedGraphContext(graph, 'selected');

    expect(context.selectedNodeId).toBe('selected');
    expect(Array.from(context.neighborNodeIds).sort()).toEqual(['neighbor-a', 'neighbor-b']);
    expect(Array.from(context.directEdgeIds).sort()).toEqual(['edge-in', 'edge-out']);
  });

  it('returns an empty context when selection is cleared', () => {
    const graph = createGraph();

    const context = buildSelectedGraphContext(graph, null);

    expect(context.selectedNodeId).toBeNull();
    expect(Array.from(context.neighborNodeIds)).toEqual([]);
    expect(Array.from(context.directEdgeIds)).toEqual([]);
  });
});
