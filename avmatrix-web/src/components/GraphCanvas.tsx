import { useEffect, useCallback, useMemo, useState, forwardRef, useImperativeHandle } from 'react';
import {
  ZoomIn,
  ZoomOut,
  Maximize2,
  Focus,
  RotateCcw,
  Play,
  Pause,
  GitBranch,
  Lightbulb,
  LightbulbOff,
} from '@/lib/lucide-icons';
import { useSigma } from '../hooks/useSigma';
import { useAppState } from '../hooks/useAppState.local-runtime';
import {
  knowledgeGraphToGraphology,
  filterGraphByDepth,
  filterGraphByLabels,
  SigmaNodeAttributes,
  SigmaEdgeAttributes,
} from '../lib/graph-adapter';
import type { GraphNode } from 'avmatrix-shared';
import { QueryFAB } from './QueryFAB';
import Graph from 'graphology';

export interface GraphCanvasHandle {
  focusNode: (nodeId: string) => void;
}

export const GraphCanvas = forwardRef<GraphCanvasHandle>((_, ref) => {
  const {
    graph,
    setSelectedNode,
    selectedNode: appSelectedNode,
    visibleLabels,
    visibleEdgeTypes,
    areGraphLinksVisible,
    depthFilter,
    highlightedNodeIds,
    setHighlightedNodeIds,
    aiCitationHighlightedNodeIds,
    aiToolHighlightedNodeIds,
    blastRadiusNodeIds,
    isAIHighlightsEnabled,
    toggleAIHighlights,
    toggleGraphLinksVisible,
    clearAIToolHighlights,
    clearAICitationHighlights,
    clearBlastRadius,
    animatedNodes,
  } = useAppState();
  const [hoveredNodeName, setHoveredNodeName] = useState<string | null>(null);

  const effectiveHighlightedNodeIds = useMemo(() => {
    if (!isAIHighlightsEnabled) return highlightedNodeIds;
    const next = new Set(highlightedNodeIds);
    for (const id of aiCitationHighlightedNodeIds) next.add(id);
    for (const id of aiToolHighlightedNodeIds) next.add(id);
    // Note: blast radius nodes are handled separately with red color
    return next;
  }, [
    highlightedNodeIds,
    aiCitationHighlightedNodeIds,
    aiToolHighlightedNodeIds,
    isAIHighlightsEnabled,
  ]);

  // Blast radius nodes (only when AI highlights enabled)
  const effectiveBlastRadiusNodeIds = useMemo(() => {
    if (!isAIHighlightsEnabled) return new Set<string>();
    return blastRadiusNodeIds;
  }, [blastRadiusNodeIds, isAIHighlightsEnabled]);

  // Animated nodes (only when AI highlights enabled)
  const effectiveAnimatedNodes = useMemo(() => {
    if (!isAIHighlightsEnabled) return new Map();
    return animatedNodes;
  }, [animatedNodes, isAIHighlightsEnabled]);

  const nodeById = useMemo(() => {
    if (!graph) return new Map<string, GraphNode>();
    return new Map(graph.nodes.map((n) => [n.id, n]));
  }, [graph]);

  const handleNodeClick = useCallback(
    (nodeId: string) => {
      if (!graph) return;
      const node = nodeById.get(nodeId);
      if (node) {
        setSelectedNode(node);
      }
    },
    [graph, nodeById, setSelectedNode],
  );

  const handleNodeHover = useCallback(
    (nodeId: string | null) => {
      if (!nodeId || !graph) {
        setHoveredNodeName(null);
        return;
      }
      const node = nodeById.get(nodeId);
      setHoveredNodeName(node ? node.properties.name : null);
    },
    [graph, nodeById],
  );

  const handleStageClick = useCallback(() => {
    setSelectedNode(null);
  }, [setSelectedNode]);

  const handleToggleAIHighlights = useCallback(() => {
    if (isAIHighlightsEnabled) {
      clearAIToolHighlights();
      clearAICitationHighlights();
      clearBlastRadius();
      setSelectedNode(null);
      setSigmaSelectedNode(null);
    }
    toggleAIHighlights();
  }, [
    isAIHighlightsEnabled,
    clearAIToolHighlights,
    clearAICitationHighlights,
    clearBlastRadius,
    setSelectedNode,
    toggleAIHighlights,
  ]);

  const {
    containerRef,
    sigmaRef,
    setGraph: setSigmaGraph,
    zoomIn,
    zoomOut,
    resetZoom,
    focusNode,
    isLayoutRunning,
    startLayout,
    stopLayout,
    selectedNode: sigmaSelectedNode,
    setSelectedNode: setSigmaSelectedNode,
  } = useSigma({
    onNodeClick: handleNodeClick,
    onNodeHover: handleNodeHover,
    onStageClick: handleStageClick,
    highlightedNodeIds: effectiveHighlightedNodeIds,
    blastRadiusNodeIds: effectiveBlastRadiusNodeIds,
    animatedNodes: effectiveAnimatedNodes,
    visibleEdgeTypes,
    areGraphLinksVisible,
  });

  // Expose focusNode to parent via ref
  useImperativeHandle(
    ref,
    () => ({
      focusNode: (nodeId: string) => {
        setSigmaSelectedNode(nodeId);
        handleNodeClick(nodeId);
      },
    }),
    [setSigmaSelectedNode, handleNodeClick],
  );

  // Update Sigma graph when KnowledgeGraph changes
  useEffect(() => {
    if (!graph) return;

    // Build communityMemberships map from MEMBER_OF relationships
    // MEMBER_OF edges: nodeId -> communityId (stored as targetId)
    const communityMemberships = new Map<string, number>();
    graph.relationships.forEach((rel) => {
      if (rel.type === 'MEMBER_OF') {
        // Find the community node to get its index
        const communityNode = nodeById.get(rel.targetId);
        if (communityNode && communityNode.label === 'Community') {
          // Extract community index from id (e.g., "comm_5" -> 5)
          const numericPart = rel.targetId.replace('comm_', '');
          const communityIdx = /^\d+$/.test(numericPart) ? parseInt(numericPart, 10) : 0;
          communityMemberships.set(rel.sourceId, communityIdx);
        }
      }
    });

    const sigmaGraph = knowledgeGraphToGraphology(graph, communityMemberships);
    setSigmaGraph(sigmaGraph);
  }, [graph, nodeById, setSigmaGraph]);

  // Update graph visibility when label filters or depth filter mode change.
  useEffect(() => {
    const sigma = sigmaRef.current;
    if (!sigma) return;

    const sigmaGraph = sigma.getGraph() as Graph<SigmaNodeAttributes, SigmaEdgeAttributes>;
    if (sigmaGraph.order === 0) return; // Don't filter empty graph

    if (depthFilter === null) {
      filterGraphByLabels(sigmaGraph, visibleLabels);
    } else {
      filterGraphByDepth(sigmaGraph, appSelectedNode?.id || null, depthFilter, visibleLabels);
    }
    sigma.refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- sigmaRef identity never changes
  }, [visibleLabels, depthFilter]);

  // Re-apply depth filtering when selection changes only if the feature is enabled.
  useEffect(() => {
    if (depthFilter === null) return;

    const sigma = sigmaRef.current;
    if (!sigma) return;

    const sigmaGraph = sigma.getGraph() as Graph<SigmaNodeAttributes, SigmaEdgeAttributes>;
    if (sigmaGraph.order === 0) return;

    filterGraphByDepth(sigmaGraph, appSelectedNode?.id || null, depthFilter, visibleLabels);
    sigma.refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- sigmaRef identity never changes
  }, [appSelectedNode?.id]);

  // Sync app selected node with sigma
  useEffect(() => {
    if (appSelectedNode) {
      setSigmaSelectedNode(appSelectedNode.id);
    } else {
      setSigmaSelectedNode(null);
    }
  }, [appSelectedNode, setSigmaSelectedNode]);

  // Focus on selected node
  const handleFocusSelected = useCallback(() => {
    if (appSelectedNode) {
      focusNode(appSelectedNode.id);
    }
  }, [appSelectedNode, focusNode]);

  // Clear selection
  const handleClearSelection = useCallback(() => {
    setSelectedNode(null);
    setSigmaSelectedNode(null);
    resetZoom();
  }, [setSelectedNode, setSigmaSelectedNode, resetZoom]);

  return (
    <div className="workspace-shell relative h-full w-full bg-workspace-base">
      <div className="pointer-events-none absolute inset-0">
        <div
          className="absolute inset-0"
          style={{
            background: `
              radial-gradient(circle at 50% 50%, rgba(154, 126, 99, 0.08) 0%, transparent 70%),
              linear-gradient(to bottom, #1f1b18, #29231f)
            `,
          }}
        />
      </div>

      <div
        ref={containerRef}
        className="sigma-container h-full w-full cursor-grab active:cursor-grabbing"
      />

      {hoveredNodeName && !sigmaSelectedNode && (
        <div className="pointer-events-none absolute top-4 left-1/2 z-20 -translate-x-1/2 animate-fade-in rounded-lg border-[2px] border-workspace-border-default bg-workspace-surface/95 px-3 py-1.5 backdrop-blur-sm">
          <span className="font-mono text-sm text-workspace-text-primary">{hoveredNodeName}</span>
        </div>
      )}

      {sigmaSelectedNode && appSelectedNode && (
        <div className="absolute top-4 left-1/2 z-20 flex -translate-x-1/2 animate-slide-up items-center gap-2 rounded-xl border-[2px] border-workspace-border-strong bg-workspace-surface px-4 py-2 backdrop-blur-sm">
          <div className="h-2 w-2 animate-pulse rounded-full bg-workspace-border-strong" />
          <span className="font-mono text-sm text-workspace-text-primary">
            {appSelectedNode.properties.name}
          </span>
          <span className="text-xs text-workspace-text-secondary">({appSelectedNode.label})</span>
          <button
            onClick={handleClearSelection}
            className="ml-2 rounded px-2 py-0.5 text-xs text-workspace-text-secondary transition-colors hover:bg-workspace-inset hover:text-workspace-text-primary"
          >
            Clear
          </button>
        </div>
      )}

      <div className="absolute right-4 bottom-4 z-10 flex flex-col gap-1">
        <button
          onClick={zoomIn}
          className="workspace-outline-button flex h-9 w-9 items-center justify-center text-workspace-text-secondary hover:text-workspace-text-primary"
          title="Zoom In"
        >
          <ZoomIn className="h-4 w-4" />
        </button>
        <button
          onClick={zoomOut}
          className="workspace-outline-button flex h-9 w-9 items-center justify-center text-workspace-text-secondary hover:text-workspace-text-primary"
          title="Zoom Out"
        >
          <ZoomOut className="h-4 w-4" />
        </button>
        <button
          onClick={resetZoom}
          className="workspace-outline-button flex h-9 w-9 items-center justify-center text-workspace-text-secondary hover:text-workspace-text-primary"
          title="Fit to Screen"
        >
          <Maximize2 className="h-4 w-4" />
        </button>

        <div className="my-1 h-px bg-workspace-border-subtle" />

        {appSelectedNode && (
          <button
            onClick={handleFocusSelected}
            className="workspace-outline-button flex h-9 w-9 items-center justify-center border-workspace-border-strong bg-workspace-surface text-workspace-text-primary"
            title="Focus on Selected Node"
          >
            <Focus className="h-4 w-4" />
          </button>
        )}

        {sigmaSelectedNode && (
          <button
            onClick={handleClearSelection}
            className="workspace-outline-button flex h-9 w-9 items-center justify-center text-workspace-text-secondary hover:text-workspace-text-primary"
            title="Clear Selection"
          >
            <RotateCcw className="h-4 w-4" />
          </button>
        )}

        <div className="my-1 h-px bg-workspace-border-subtle" />

        <button
          onClick={isLayoutRunning ? stopLayout : startLayout}
          className={`flex h-9 w-9 items-center justify-center rounded-md border transition-all ${
            isLayoutRunning
              ? 'animate-pulse border-workspace-border-strong bg-workspace-surface text-workspace-text-primary'
              : 'border-workspace-border-default bg-workspace-surface text-workspace-text-secondary hover:bg-workspace-inset hover:text-workspace-text-primary'
          } `}
          title={isLayoutRunning ? 'Stop Layout' : 'Run Layout Again'}
        >
          {isLayoutRunning ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
        </button>
      </div>

      {isLayoutRunning && (
        <div className="absolute bottom-4 left-1/2 z-10 flex -translate-x-1/2 animate-fade-in items-center gap-2 rounded-full border-[2px] border-workspace-border-default bg-workspace-surface px-3 py-1.5 backdrop-blur-sm">
          <div className="h-2 w-2 animate-ping rounded-full bg-workspace-border-strong" />
          <span className="text-xs font-medium text-workspace-text-primary">
            Layout optimizing...
          </span>
        </div>
      )}

      <QueryFAB />

      <div className="absolute top-4 right-4 z-20 flex flex-col gap-2">
        <button
          type="button"
          onClick={handleToggleAIHighlights}
          className={
            isAIHighlightsEnabled
              ? 'flex h-10 w-10 items-center justify-center rounded-lg border-[2px] border-workspace-border-strong bg-workspace-surface text-workspace-text-primary transition-colors hover:bg-workspace-inset'
              : 'flex h-10 w-10 items-center justify-center rounded-lg border-[2px] border-workspace-border-default bg-workspace-surface text-workspace-text-secondary transition-colors hover:bg-workspace-inset hover:text-workspace-text-primary'
          }
          title={
            isAIHighlightsEnabled ? 'Turn off AI-driven highlights' : 'Turn on AI-driven highlights'
          }
          aria-label={
            isAIHighlightsEnabled ? 'Turn off AI-driven highlights' : 'Turn on AI-driven highlights'
          }
          data-testid="ai-highlights-toggle"
        >
          {isAIHighlightsEnabled ? (
            <Lightbulb className="h-4 w-4" />
          ) : (
            <LightbulbOff className="h-4 w-4" />
          )}
        </button>

        <button
          type="button"
          onClick={toggleGraphLinksVisible}
          aria-pressed={areGraphLinksVisible}
          className={
            areGraphLinksVisible
              ? 'flex h-10 w-10 items-center justify-center rounded-lg border-[2px] border-workspace-border-strong bg-workspace-surface text-workspace-text-primary transition-colors hover:bg-workspace-inset'
              : 'flex h-10 w-10 items-center justify-center rounded-lg border-[2px] border-workspace-border-default bg-workspace-surface text-workspace-text-secondary transition-colors hover:bg-workspace-inset hover:text-workspace-text-primary'
          }
          title={areGraphLinksVisible ? 'Turn off all graph links' : 'Turn on all graph links'}
          aria-label={areGraphLinksVisible ? 'Turn off all graph links' : 'Turn on all graph links'}
          data-testid="graph-links-toggle"
        >
          <GitBranch className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
});

GraphCanvas.displayName = 'GraphCanvas';
