import { describe, expect, it } from 'vitest';
import { getSelectedContextEdgeSize } from '../../src/lib/graph-edge-render-style';

describe('graph edge render style', () => {
  it('keeps current selected-context thickness when ambient graph links are on', () => {
    expect(
      getSelectedContextEdgeSize({
        baseSize: 1,
        areAmbientGraphLinksVisible: true,
      }),
    ).toBe(4);
  });

  it('reduces selected-context thickness to 70 percent when ambient graph links are off', () => {
    expect(
      getSelectedContextEdgeSize({
        baseSize: 1,
        areAmbientGraphLinksVisible: false,
      }),
    ).toBeCloseTo(2.8);
  });

  it('still scales the minimum selected-context width down when ambient graph links are off', () => {
    expect(
      getSelectedContextEdgeSize({
        baseSize: 0.4,
        areAmbientGraphLinksVisible: false,
      }),
    ).toBeCloseTo(2.1);
  });
});
