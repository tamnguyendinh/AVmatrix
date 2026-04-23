import { createContext, useContext, useCallback, useMemo, useState, ReactNode } from 'react';
import type { GraphNode, NodeLabel } from 'avmatrix-shared';
import type { KnowledgeGraph } from '../../core/graph/types';
import { DEFAULT_VISIBLE_LABELS, DEFAULT_VISIBLE_EDGES, type EdgeType } from '../../lib/constants';
import {
  readGraphLinksVisibilityPreference,
  writeGraphLinksVisibilityPreference,
} from '../../lib/graph-links-visibility';

interface GraphStateContextValue {
  graph: KnowledgeGraph | null;
  setGraph: (graph: KnowledgeGraph | null) => void;
  selectedNode: GraphNode | null;
  setSelectedNode: (node: GraphNode | null) => void;
  visibleLabels: NodeLabel[];
  toggleLabelVisibility: (label: NodeLabel) => void;
  visibleEdgeTypes: EdgeType[];
  toggleEdgeVisibility: (edgeType: EdgeType) => void;
  areGraphLinksVisible: boolean;
  setGraphLinksVisible: (visible: boolean) => void;
  toggleGraphLinksVisible: () => void;
  depthFilter: number | null;
  setDepthFilter: (depth: number | null) => void;
  highlightedNodeIds: Set<string>;
  setHighlightedNodeIds: (ids: Set<string>) => void;
}

const GraphStateContext = createContext<GraphStateContextValue | null>(null);

export const GraphStateProvider = ({ children }: { children: ReactNode }) => {
  const [graph, setGraph] = useState<KnowledgeGraph | null>(null);
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [visibleLabels, setVisibleLabels] = useState<NodeLabel[]>(DEFAULT_VISIBLE_LABELS);
  const [visibleEdgeTypes, setVisibleEdgeTypes] = useState<EdgeType[]>(DEFAULT_VISIBLE_EDGES);
  const [areGraphLinksVisible, setGraphLinksVisibleState] = useState<boolean>(() =>
    readGraphLinksVisibilityPreference(),
  );
  const [depthFilter, setDepthFilter] = useState<number | null>(null);
  const [highlightedNodeIds, setHighlightedNodeIds] = useState<Set<string>>(new Set());

  const toggleLabelVisibility = useCallback((label: NodeLabel) => {
    setVisibleLabels((prev) =>
      prev.includes(label) ? prev.filter((l) => l !== label) : [...prev, label],
    );
  }, []);

  const toggleEdgeVisibility = useCallback((edgeType: EdgeType) => {
    setVisibleEdgeTypes((prev) =>
      prev.includes(edgeType) ? prev.filter((e) => e !== edgeType) : [...prev, edgeType],
    );
  }, []);

  const setGraphLinksVisible = useCallback((visible: boolean) => {
    setGraphLinksVisibleState(visible);
    writeGraphLinksVisibilityPreference(visible);
  }, []);

  const toggleGraphLinksVisible = useCallback(() => {
    setGraphLinksVisibleState((prev) => {
      const next = !prev;
      writeGraphLinksVisibilityPreference(next);
      return next;
    });
  }, []);

  const value = useMemo<GraphStateContextValue>(
    () => ({
      graph,
      setGraph,
      selectedNode,
      setSelectedNode,
      visibleLabels,
      toggleLabelVisibility,
      visibleEdgeTypes,
      toggleEdgeVisibility,
      areGraphLinksVisible,
      setGraphLinksVisible,
      toggleGraphLinksVisible,
      depthFilter,
      setDepthFilter,
      highlightedNodeIds,
      setHighlightedNodeIds,
    }),
    [
      graph,
      selectedNode,
      visibleLabels,
      visibleEdgeTypes,
      areGraphLinksVisible,
      setGraphLinksVisible,
      toggleGraphLinksVisible,
      depthFilter,
      highlightedNodeIds,
    ],
  );

  return <GraphStateContext.Provider value={value}>{children}</GraphStateContext.Provider>;
};

export const useGraphState = (): GraphStateContextValue => {
  const ctx = useContext(GraphStateContext);
  if (!ctx) {
    throw new Error('useGraphState must be used within a GraphStateProvider');
  }
  return ctx;
};
