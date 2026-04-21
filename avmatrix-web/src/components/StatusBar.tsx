import { useMemo } from 'react';
import { useAppState } from '../hooks/useAppState.local-runtime';

export const StatusBar = () => {
  const { graph, progress } = useAppState();

  const nodeCount = graph?.nodes.length ?? 0;
  const edgeCount = graph?.relationships.length ?? 0;

  // Detect primary language
  const primaryLanguage = useMemo(() => {
    if (!graph) return null;
    const languages = graph.nodes.map((n) => n.properties.language).filter(Boolean);
    if (languages.length === 0) return null;

    const counts = languages.reduce(
      (acc, lang) => {
        acc[lang!] = (acc[lang!] || 0) + 1;
        return acc;
      },
      {} as Record<string, number>,
    );

    return Object.entries(counts).sort((a, b) => b[1] - a[1])[0]?.[0];
  }, [graph]);

  return (
    <footer className="flex items-center justify-between border-t border-dashed border-border-subtle bg-deep px-5 py-2 text-[11px] text-text-muted">
      {/* Left - Status */}
      <div className="flex items-center gap-4">
        {progress && progress.phase !== 'complete' ? (
          <>
            <div className="h-1 w-28 overflow-hidden rounded-full bg-elevated">
              <div
                className="h-full rounded-full bg-gradient-to-r from-accent to-node-interface transition-all duration-300"
                style={{ width: `${progress.percent}%` }}
              />
            </div>
            <span>{progress.message}</span>
          </>
        ) : (
          <div className="flex items-center gap-1.5" data-testid="status-ready">
            <span className="h-1.5 w-1.5 rounded-full bg-node-function" />
            <span>Ready</span>
          </div>
        )}
      </div>

      {/* Center spacer - keeps footer balance without external CTA */}
      <div aria-hidden="true" className="h-6 w-[220px] shrink-0" />

      {/* Right - Stats */}
      <div className="flex items-center gap-3" data-testid="graph-stats">
        {graph && (
          <>
            <span>{nodeCount} nodes</span>
            <span className="text-border-default">•</span>
            <span>{edgeCount} edges</span>
            {primaryLanguage && (
              <>
                <span className="text-border-default">•</span>
                <span>{primaryLanguage}</span>
              </>
            )}
          </>
        )}
      </div>
    </footer>
  );
};
