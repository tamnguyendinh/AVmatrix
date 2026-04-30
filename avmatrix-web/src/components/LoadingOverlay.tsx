import type { PipelineProgress } from 'avmatrix-shared';
import { EncouragementLine } from './EncouragementLine';
import { useAppState } from '../hooks/useAppState.local-runtime';

interface LoadingOverlayProps {
  progress: PipelineProgress;
}

export const LoadingOverlay = ({ progress }: LoadingOverlayProps) => {
  const { projectName } = useAppState();
  const isGraphDownload = progress.message === 'Loading graph...';
  const showPercent = !isGraphDownload && progress.showPercent !== false;
  const repoLabel = progress.targetRepoName || projectName || 'AVmatrix';

  return (
    <div className="press-shell press-ruled fixed inset-0 z-50 flex flex-col items-center justify-center">
      <div className="relative mb-10">
        <div className="max-w-[min(28rem,calc(100vw-3rem))] border-[2px] border-border-subtle bg-surface px-5 py-3 shadow-[var(--shadow-dropdown)]">
          <span className="font-mono text-sm font-semibold text-text-primary">
            [ <span className="inline-block max-w-[20rem] truncate align-bottom">{repoLabel}</span> ]
          </span>
        </div>
      </div>

      <div className="mb-4 w-80">
        <div className="h-1.5 overflow-hidden rounded-full bg-inset">
          <div
            className={`h-full rounded-full bg-border-strong transition-all duration-300 ease-out ${
              showPercent ? '' : 'w-full animate-pulse opacity-50'
            }`}
            style={showPercent ? { width: `${progress.percent}%` } : undefined}
          />
        </div>
      </div>

      <div className="text-center">
        <p className="press-eyebrow mb-1 text-text-secondary">
          {progress.message}
          <span className="animate-pulse">|</span>
        </p>
        {progress.detail && (
          <p className="max-w-md truncate font-reading text-xs text-text-secondary">{progress.detail}</p>
        )}
        <p className="mt-3 font-reading text-sm text-text-secondary">
          This may take a moment for large repositories
        </p>
      </div>

      <div className="mt-5 w-80 max-w-[calc(100vw-3rem)]">
        <div className="h-1.5 overflow-hidden rounded-full bg-inset">
          <div className="h-full w-full rounded-full bg-border-strong opacity-50" />
        </div>
      </div>

      {progress.stats && (
        <div className="mt-8 flex items-center gap-6 font-mono text-xs text-text-secondary">
          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-border-default" />
            <span>
              {progress.stats.filesProcessed} / {progress.stats.totalFiles} files
            </span>
          </div>
          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-border-strong" />
            <span>{progress.stats.nodesCreated} nodes</span>
          </div>
        </div>
      )}

      <div className="mt-6">
        <EncouragementLine />
      </div>
    </div>
  );
};
