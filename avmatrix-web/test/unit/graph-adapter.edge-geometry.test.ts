import { describe, expect, it } from 'vitest';
import { createKnowledgeGraph } from '../../src/core/graph/graph';
import { knowledgeGraphToGraphology } from '../../src/lib/graph-adapter';
import { createCallsRelationship, createFileNode, createFunctionNode } from '../fixtures/graph';

describe('knowledgeGraphToGraphology edge geometry', () => {
  it('creates straight edges without curved-edge metadata', () => {
    const graph = createKnowledgeGraph();
    const fileNode = createFileNode('index.ts', 'src/index.ts');
    const functionNode = createFunctionNode('main', 'src/index.ts', 1);

    graph.addNode(fileNode);
    graph.addNode(functionNode);
    graph.addRelationship(createCallsRelationship(fileNode.id, functionNode.id));

    const sigmaGraph = knowledgeGraphToGraphology(graph);
    const [edgeId] = sigmaGraph.edges();
    const edgeAttributes = sigmaGraph.getEdgeAttributes(edgeId) as Record<string, unknown>;

    expect(edgeId).toBeDefined();
    expect(edgeAttributes.size).toBeGreaterThan(0);
    expect(edgeAttributes).not.toHaveProperty('type');
    expect(edgeAttributes).not.toHaveProperty('curvature');
  });
});
