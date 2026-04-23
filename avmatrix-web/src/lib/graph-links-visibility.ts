import type { EdgeType } from './constants';

export const GRAPH_LINKS_VISIBILITY_STORAGE_KEY = 'avmatrix.graphLinksVisible';

export const readGraphLinksVisibilityPreference = (): boolean => {
  if (typeof window === 'undefined') return true;

  try {
    const saved = window.localStorage.getItem(GRAPH_LINKS_VISIBILITY_STORAGE_KEY);
    if (saved === null) return true;
    return saved !== 'false';
  } catch {
    return true;
  }
};

export const writeGraphLinksVisibilityPreference = (visible: boolean): void => {
  if (typeof window === 'undefined') return;

  try {
    window.localStorage.setItem(GRAPH_LINKS_VISIBILITY_STORAGE_KEY, String(visible));
  } catch {
    // Best-effort persistence only.
  }
};

export const shouldHideGraphEdge = ({
  areGraphLinksVisible,
  relationType,
  visibleEdgeTypes,
}: {
  areGraphLinksVisible: boolean;
  relationType?: string;
  visibleEdgeTypes: EdgeType[] | null;
}): boolean => {
  if (!areGraphLinksVisible) {
    return true;
  }

  if (visibleEdgeTypes && relationType) {
    return !visibleEdgeTypes.includes(relationType as EdgeType);
  }

  return false;
};
