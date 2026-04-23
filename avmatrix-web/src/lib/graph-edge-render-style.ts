const SELECTED_CONTEXT_EDGE_MIN_SIZE = 3;
const SELECTED_CONTEXT_EDGE_SIZE_MULTIPLIER = 4;
const FOCUSED_SELECTED_CONTEXT_EDGE_SCALE = 0.7;

export const getSelectedContextEdgeSize = ({
  baseSize,
  areAmbientGraphLinksVisible,
}: {
  baseSize: number;
  areAmbientGraphLinksVisible: boolean;
}): number => {
  const selectedContextSize = Math.max(
    SELECTED_CONTEXT_EDGE_MIN_SIZE,
    baseSize * SELECTED_CONTEXT_EDGE_SIZE_MULTIPLIER,
  );

  if (areAmbientGraphLinksVisible) {
    return selectedContextSize;
  }

  return selectedContextSize * FOCUSED_SELECTED_CONTEXT_EDGE_SCALE;
};
