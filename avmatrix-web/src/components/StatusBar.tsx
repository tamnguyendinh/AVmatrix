import { useMemo } from 'react';
import { useAppState } from '../hooks/useAppState.local-runtime';

export const StatusBar = () => {
  const { graph, progress } = useAppState();

  const nodeCount = graph?.nodes.length ?? 0;
  const edgeCount = graph?.relationships.length ?? 0;

  // Detect primary language
  const primaryLanguage = useMemo(() => {
    if (!graph) return null;
    const languages = graph.nodes
      .map((n) => n.properties.language)
      .filter(
        (language): language is string => typeof language === 'string' && language.length > 0,
      );
    if (languages.length === 0) return null;

    const counts = languages.reduce(
      (acc, lang) => {
        acc[lang] = (acc[lang] || 0) + 1;
        return acc;
      },
      {} as Record<string, number>,
    );

    return Object.entries(counts).sort((a, b) => b[1] - a[1])[0]?.[0];
  }, [graph]);

  return (
    <footer className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 border-t-[3px] border-border-default bg-surface px-5 py-3 text-[11px] text-text-secondary">
      <div className="flex min-w-0 items-center gap-4">
        {progress && progress.phase !== 'complete' ? (
          <>
            <div className="h-1 w-28 overflow-hidden rounded-full bg-inset">
              <div
                className="h-full rounded-full bg-border-strong transition-all duration-300"
                style={{ width: `${progress.percent}%` }}
              />
            </div>
            <span className="font-reading">{progress.message}</span>
          </>
        ) : (
          <div
            className="press-eyebrow flex items-center gap-1.5 text-green-500"
            data-testid="status-ready"
          >
            <span className="h-1.5 w-1.5 rounded-full bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.8)]" />
            <span>Ready</span>
          </div>
        )}
      </div>

      <div
        className="flex flex-wrap items-center justify-end gap-3 font-mono text-xs font-semibold text-text-primary"
        data-testid="graph-stats"
      >
        {graph && (
          <>
            <span>{nodeCount} nodes</span>
            <span className="text-border-strong">•</span>
            <span>{edgeCount} edges</span>
            {primaryLanguage && (
              <>
                <span className="text-border-strong">•</span>
                <span>{primaryLanguage}</span>
              </>
            )}
          </>
        )}
      </div>
    </footer>
  );
};
