import { describe, expect, it } from 'vitest';
import { getGraphEdgeVisibilityMode } from '../../src/lib/graph-edge-visibility-mode';

describe('graph edge visibility mode', () => {
  it('shows ambient edges when ambient links are enabled and nothing is selected', () => {
    expect(
      getGraphEdgeVisibilityMode({
        areAmbientGraphLinksVisible: true,
        currentSelectedNodeId: null,
        sourceNodeId: 'a',
        targetNodeId: 'b',
        relationType: 'CALLS',
        visibleEdgeTypes: ['CALLS'],
      }),
    ).toBe('ambient');
  });

  it('hides non-context edges when ambient links are disabled', () => {
    expect(
      getGraphEdgeVisibilityMode({
        areAmbientGraphLinksVisible: false,
        currentSelectedNodeId: 'selected',
        sourceNodeId: 'a',
        targetNodeId: 'b',
        relationType: 'CALLS',
        visibleEdgeTypes: ['CALLS'],
      }),
    ).toBe('hidden');
  });

  it('keeps direct selected-node context edges visible when ambient links are disabled', () => {
    expect(
      getGraphEdgeVisibilityMode({
        areAmbientGraphLinksVisible: false,
        currentSelectedNodeId: 'selected',
        sourceNodeId: 'selected',
        targetNodeId: 'neighbor',
        relationType: 'CALLS',
        visibleEdgeTypes: ['CALLS'],
      }),
    ).toBe('selected-context');
  });

  it('still respects edge type filters for selected-node context edges', () => {
    expect(
      getGraphEdgeVisibilityMode({
        areAmbientGraphLinksVisible: false,
        currentSelectedNodeId: 'selected',
        sourceNodeId: 'selected',
        targetNodeId: 'neighbor',
        relationType: 'CALLS',
        visibleEdgeTypes: ['IMPORTS'],
      }),
    ).toBe('hidden');
  });

  it('removes selected-node context visibility once the selection is cleared', () => {
    expect(
      getGraphEdgeVisibilityMode({
        areAmbientGraphLinksVisible: false,
        currentSelectedNodeId: null,
        sourceNodeId: 'selected',
        targetNodeId: 'neighbor',
        relationType: 'CALLS',
        visibleEdgeTypes: ['CALLS'],
      }),
    ).toBe('hidden');
  });
});
