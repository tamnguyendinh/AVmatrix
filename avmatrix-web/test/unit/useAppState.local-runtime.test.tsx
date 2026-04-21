import { act, render, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReactNode } from 'react';
import { createKnowledgeGraph } from '../../src/core/graph/graph';
import type { GraphNode } from 'avmatrix-shared';
import { setBackendUrl } from '../../src/services/backend-client';
import { AppStateProvider, useAppState } from '../../src/hooks/useAppState.local-runtime';

let appState: ReturnType<typeof useAppState> | null = null;

function Harness() {
  appState = useAppState();
  return null;
}

function renderHarness(children?: ReactNode) {
  return render(
    <AppStateProvider>
      <Harness />
      {children}
    </AppStateProvider>,
  );
}

const createFileNode = (id: string, filePath: string): GraphNode => ({
  id,
  label: 'File',
  properties: {
    name: filePath.split('/').pop() ?? filePath,
    filePath,
  },
});

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

describe('useAppState.local-runtime', () => {
  beforeEach(() => {
    appState = null;
    setBackendUrl('http://localhost:4747');
    vi.stubGlobal('requestAnimationFrame', ((callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    }) as typeof requestAnimationFrame);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('exposes a stable chat runtime bridge with current repo identity', async () => {
    renderHarness();
    await waitFor(() => expect(appState).not.toBeNull());

    expect(appState!.chatRuntimeBridge.getRepoName()).toBeUndefined();
    expect(appState!.chatRuntimeBridge.getEmbeddingStatus()).toBe('idle');

    await act(async () => {
      appState!.setProjectName('avmatrix');
    });
    expect(appState!.chatRuntimeBridge.getRepoName()).toBe('avmatrix');

    await act(async () => {
      appState!.setCurrentRepo('website');
    });
    expect(appState!.chatRuntimeBridge.getRepoName()).toBe('website');
  });

  it('routes grounding references through the chat runtime bridge into code references', async () => {
    renderHarness();
    await waitFor(() => expect(appState).not.toBeNull());

    const graph = createKnowledgeGraph();
    graph.addNode(createFileNode('File:src/foo.ts', 'src/foo.ts'));
    graph.addNode(
      createFunctionNode('Function:src/foo.ts:loadFoo', 'loadFoo', 'src/foo.ts', 10, 20),
    );

    await act(async () => {
      appState!.setGraph(graph);
    });

    await act(async () => {
      appState!.chatRuntimeBridge.handleContentGrounding(
        'Inspect [[src/foo.ts:4-6]] and [[Function:loadFoo]]',
      );
    });

    expect(appState!.codeReferences).toHaveLength(2);
    expect(appState!.isCodePanelOpen).toBe(true);
    expect(appState!.codeReferences).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          filePath: 'src/foo.ts',
          startLine: 3,
          endLine: 5,
          label: 'File',
          source: 'ai',
        }),
        expect.objectContaining({
          filePath: 'src/foo.ts',
          startLine: 9,
          endLine: 19,
          label: 'Function',
          name: 'loadFoo',
          source: 'ai',
        }),
      ]),
    );
  });

  it('applies highlight and impact markers through the chat runtime bridge', async () => {
    renderHarness();
    await waitFor(() => expect(appState).not.toBeNull());

    const graph = createKnowledgeGraph();
    graph.addNode(createFileNode('File:src/foo.ts', 'src/foo.ts'));
    graph.addNode(
      createFunctionNode('Function:src/foo.ts:loadFoo', 'loadFoo', 'src/foo.ts', 10, 20),
    );
    graph.addNode(
      createFunctionNode('Function:src/foo.ts:saveFoo', 'saveFoo', 'src/foo.ts', 30, 40),
    );

    await act(async () => {
      appState!.setGraph(graph);
    });

    await act(async () => {
      appState!.chatRuntimeBridge.handleToolResult(
        [
          '[HIGHLIGHT_NODES:Function:src/foo.ts:loadFoo]',
          '[IMPACT:Function:src/foo.ts:saveFoo]',
        ].join(' '),
      );
    });

    expect(Array.from(appState!.aiToolHighlightedNodeIds)).toEqual([
      'Function:src/foo.ts:loadFoo',
    ]);
    expect(Array.from(appState!.blastRadiusNodeIds)).toEqual([
      'Function:src/foo.ts:saveFoo',
    ]);
  });
});
