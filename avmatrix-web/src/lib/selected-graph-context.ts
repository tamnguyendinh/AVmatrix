import Graph from 'graphology';
import type { SigmaNodeAttributes, SigmaEdgeAttributes } from './graph-adapter';

export interface SelectedGraphContext {
  selectedNodeId: string | null;
  neighborNodeIds: Set<string>;
  directEdgeIds: Set<string>;
}

export const buildSelectedGraphContext = (
  graph: Graph<SigmaNodeAttributes, SigmaEdgeAttributes> | null,
  selectedNodeId: string | null,
): SelectedGraphContext => {
  if (!graph || !selectedNodeId || !graph.hasNode(selectedNodeId)) {
    return {
      selectedNodeId: null,
      neighborNodeIds: new Set<string>(),
      directEdgeIds: new Set<string>(),
    };
  }

  const neighborNodeIds = new Set<string>();
  const directEdgeIds = new Set<string>();

  graph.forEachNeighbor(selectedNodeId, (neighborNodeId) => {
    neighborNodeIds.add(neighborNodeId);
  });

  graph.forEachEdge(selectedNodeId, (edgeId) => {
    directEdgeIds.add(edgeId);
  });

  return {
    selectedNodeId,
    neighborNodeIds,
    directEdgeIds,
  };
};
