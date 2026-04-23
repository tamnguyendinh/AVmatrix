import { act, render, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createKnowledgeGraph } from '../../src/core/graph/graph';
import type { GraphNode } from 'avmatrix-shared';
import { AppStateProvider, useAppState } from '../../src/hooks/useAppState.local-runtime';

const {
  filterGraphByDepthSpy,
  filterGraphByLabelsSpy,
  sigmaRefreshSpy,
  setSigmaSelectedNodeSpy,
  setSigmaGraphSpy,
  sigmaGraph,
} = vi.hoisted(() => {
  return {
    filterGraphByDepthSpy: vi.fn(),
    filterGraphByLabelsSpy: vi.fn(),
    sigmaRefreshSpy: vi.fn(),
    setSigmaSelectedNodeSpy: vi.fn(),
    setSigmaGraphSpy: vi.fn(),
    sigmaGraph: { order: 1 },
  };
});

vi.mock('../../src/components/QueryFAB', () => ({
  QueryFAB: () => null,
}));

vi.mock('../../src/hooks/useSigma', () => ({
  useSigma: () => ({
    containerRef: { current: null },
    sigmaRef: {
      current: {
        getGraph: () => sigmaGraph,
        refresh: sigmaRefreshSpy,
      },
    },
    setGraph: setSigmaGraphSpy,
    zoomIn: vi.fn(),
    zoomOut: vi.fn(),
    resetZoom: vi.fn(),
    focusNode: vi.fn(),
    isLayoutRunning: false,
    startLayout: vi.fn(),
    stopLayout: vi.fn(),
    selectedNode: null,
    setSelectedNode: setSigmaSelectedNodeSpy,
    refreshHighlights: vi.fn(),
  }),
}));

vi.mock('../../src/lib/graph-adapter', () => ({
  knowledgeGraphToGraphology: vi.fn(() => sigmaGraph),
  filterGraphByDepth: filterGraphByDepthSpy,
  filterGraphByLabels: filterGraphByLabelsSpy,
}));

import { GraphCanvas } from '../../src/components/GraphCanvas';

let appState: ReturnType<typeof useAppState> | null = null;

function Harness() {
  appState = useAppState();
  return null;
}

const createFunctionNode = (
  id: string,
  name: string,
  filePath: string,
  startLine: number,
  endLine: number,
): GraphNode => ({
  id,
  label: 'Function',
  properties: {
    name,
    filePath,
    startLine,
    endLine,
  },
});

describe('GraphCanvas selection performance guards', () => {
  beforeEach(() => {
    appState = null;
    filterGraphByDepthSpy.mockReset();
    filterGraphByLabelsSpy.mockReset();
    sigmaRefreshSpy.mockReset();
    setSigmaSelectedNodeSpy.mockReset();
    setSigmaGraphSpy.mockReset();
  });

  it('does not re-run full graph filtering when selection changes and depthFilter is null', async () => {
    render(
      <AppStateProvider>
        <Harness />
        <GraphCanvas />
      </AppStateProvider>,
    );

    await waitFor(() => expect(appState).not.toBeNull());

    const graph = createKnowledgeGraph();
    const selectedNode = createFunctionNode(
      'Function:src/foo.ts:loadFoo',
      'loadFoo',
      'src/foo.ts',
      10,
      20,
    );
    graph.addNode(selectedNode);

    await act(async () => {
      appState!.setGraph(graph);
    });

    filterGraphByDepthSpy.mockClear();
    filterGraphByLabelsSpy.mockClear();
    sigmaRefreshSpy.mockClear();

    await act(async () => {
      appState!.setSelectedNode(selectedNode);
    });

    expect(filterGraphByDepthSpy).not.toHaveBeenCalled();
    expect(filterGraphByLabelsSpy).not.toHaveBeenCalled();
  });

  it('still re-applies depth filtering when selection changes and depthFilter is enabled', async () => {
    render(
      <AppStateProvider>
        <Harness />
        <GraphCanvas />
      </AppStateProvider>,
    );

    await waitFor(() => expect(appState).not.toBeNull());

    const graph = createKnowledgeGraph();
    const selectedNode = createFunctionNode(
      'Function:src/foo.ts:loadFoo',
      'loadFoo',
      'src/foo.ts',
      10,
      20,
    );
    graph.addNode(selectedNode);

    await act(async () => {
      appState!.setGraph(graph);
      appState!.setDepthFilter(2);
    });

    filterGraphByDepthSpy.mockClear();
    filterGraphByLabelsSpy.mockClear();
    sigmaRefreshSpy.mockClear();

    await act(async () => {
      appState!.setSelectedNode(selectedNode);
    });

    expect(filterGraphByDepthSpy).toHaveBeenCalledWith(
      sigmaGraph,
      selectedNode.id,
      2,
      expect.any(Array),
    );
  });
});
