import type { EdgeType } from './constants';

export type GraphEdgeVisibilityMode = 'hidden' | 'ambient' | 'selected-context';

const isEdgeTypeVisible = ({
  relationType,
  visibleEdgeTypes,
}: {
  relationType?: string;
  visibleEdgeTypes: EdgeType[] | null;
}): boolean => {
  if (!visibleEdgeTypes || !relationType) {
    return true;
  }

  return visibleEdgeTypes.includes(relationType as EdgeType);
};

export const getGraphEdgeVisibilityMode = ({
  areAmbientGraphLinksVisible,
  currentSelectedNodeId,
  sourceNodeId,
  targetNodeId,
  relationType,
  visibleEdgeTypes,
}: {
  areAmbientGraphLinksVisible: boolean;
  currentSelectedNodeId: string | null;
  sourceNodeId: string;
  targetNodeId: string;
  relationType?: string;
  visibleEdgeTypes: EdgeType[] | null;
}): GraphEdgeVisibilityMode => {
  if (
    !isEdgeTypeVisible({
      relationType,
      visibleEdgeTypes,
    })
  ) {
    return 'hidden';
  }

  if (
    currentSelectedNodeId &&
    (sourceNodeId === currentSelectedNodeId || targetNodeId === currentSelectedNodeId)
  ) {
    return 'selected-context';
  }

  if (areAmbientGraphLinksVisible) {
    return 'ambient';
  }

  return 'hidden';
};
