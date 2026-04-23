import { describe, expect, it } from 'vitest';
import { shouldHideGraphEdge } from '../../src/lib/graph-links-visibility';

describe('ambient graph links visibility helper', () => {
  it('hides ambient edges when the ambient toggle is off', () => {
    expect(
      shouldHideGraphEdge({
        areGraphLinksVisible: false,
        relationType: 'CALLS',
        visibleEdgeTypes: ['CALLS'],
      }),
    ).toBe(true);
  });

  it('still respects visible edge type filters when the master toggle is on', () => {
    expect(
      shouldHideGraphEdge({
        areGraphLinksVisible: true,
        relationType: 'CALLS',
        visibleEdgeTypes: ['CALLS', 'IMPORTS'],
      }),
    ).toBe(false);

    expect(
      shouldHideGraphEdge({
        areGraphLinksVisible: true,
        relationType: 'EXTENDS',
        visibleEdgeTypes: ['CALLS', 'IMPORTS'],
      }),
    ).toBe(true);
  });

  it('does not hide edges without a typed relation when links are globally enabled', () => {
    expect(
      shouldHideGraphEdge({
        areGraphLinksVisible: true,
        visibleEdgeTypes: ['CALLS'],
      }),
    ).toBe(false);
  });
});
