import type { PipelineProgress } from 'avmatrix-shared';

interface LoadingOverlayProps {
  progress: PipelineProgress;
}

export const LoadingOverlay = ({ progress }: LoadingOverlayProps) => {
  const isGraphDownload = progress.message === 'Downloading graph...';
  const showPercent = !isGraphDownload && progress.showPercent !== false;

  return (
    <div className="press-shell press-ruled fixed inset-0 z-50 flex flex-col items-center justify-center">
      <div className="relative mb-10">
        <div className="flex h-28 w-28 items-center justify-center rounded-full border-[3px] border-border-strong bg-surface">
          <span className="press-title text-4xl">A</span>
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
    </div>
  );
};
