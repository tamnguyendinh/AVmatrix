/**
 * RepoLanding
 *
 * Unified landing screen shown when the backend is connected and at least one
 * repository is indexed. Displays pre-indexed repos as selectable cards, plus
 * an "Analyze a New Repository" section powered by RepoAnalyzer.
 *
 * Rendering context:
 *   DropZone (Crossfade, phase="landing")
 *     └─ RepoLanding
 *          ├─ RepoCard (× N)
 *          └─ RepoAnalyzer (variant="onboarding")
 */

import { useState } from 'react';
import { ArrowRight, GitBranch, FileCode, Layers, Loader2, X } from '@/lib/lucide-icons';
import { RepoAnalyzer } from './RepoAnalyzer';
import type { BackendRepo } from '../services/backend-client';

// ── Helpers ──────────────────────────────────────────────────────────────────

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60_000);
  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 30) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

// ── Repo card ────────────────────────────────────────────────────────────────

function RepoCard({
  repo,
  onClick,
  onRemove,
  isRemoving,
}: {
  repo: BackendRepo;
  onClick: () => void;
  onRemove?: () => void;
  isRemoving?: boolean;
}) {
  const stats = repo.stats;

  return (
    <div className="press-panel group relative w-full transition-all duration-200 hover:border-border-strong">
      <button
        onClick={onClick}
        disabled={isRemoving}
        data-testid="landing-repo-card"
        className="w-full cursor-pointer p-4 pr-12 text-left disabled:cursor-wait"
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <GitBranch className="h-4 w-4 shrink-0 text-border-strong" />
              <h3 className="truncate font-mono text-sm font-semibold text-text-primary transition-colors group-hover:text-border-strong">
                {repo.name}
              </h3>
            </div>
            {repo.indexedAt && (
              <p className="mt-1 pl-6 text-xs text-text-muted">
                Indexed {formatRelativeTime(repo.indexedAt)}
              </p>
            )}
          </div>
          <ArrowRight className="mr-3 h-4 w-4 shrink-0 text-text-muted opacity-0 transition-all duration-200 group-hover:translate-x-0.5 group-hover:text-border-strong group-hover:opacity-100" />
        </div>

        {stats && (stats.files || stats.nodes) && (
          <div className="mt-3 flex flex-wrap gap-2 pl-6">
            {stats.files != null && (
              <span className="press-badge inline-flex items-center gap-1 border-border-default bg-base px-2 py-0.5 text-[11px] text-text-secondary normal-case tracking-normal">
                <FileCode className="h-3 w-3" /> {stats.files.toLocaleString()} files
              </span>
            )}
            {stats.nodes != null && (
              <span className="press-badge inline-flex items-center gap-1 border-border-default bg-base px-2 py-0.5 text-[11px] text-text-secondary normal-case tracking-normal">
                <Layers className="h-3 w-3" /> {stats.nodes.toLocaleString()} symbols
              </span>
            )}
            {stats.processes != null && stats.processes > 0 && (
              <span className="press-badge inline-flex items-center border-border-default bg-base px-2 py-0.5 text-[11px] text-text-secondary normal-case tracking-normal">
                {stats.processes} flows
              </span>
            )}
          </div>
        )}
      </button>

      {onRemove && (
        <button
          type="button"
          onClick={(event) => {
            event.stopPropagation();
            onRemove();
          }}
          disabled={isRemoving}
          className="absolute top-3 right-3 cursor-pointer rounded p-1 text-text-muted opacity-70 transition-all hover:bg-base hover:text-red-400 hover:opacity-100 disabled:cursor-wait disabled:text-border-strong"
          aria-label={`Remove ${repo.name} from repository list`}
          title={`Remove ${repo.name}`}
        >
          {isRemoving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <X className="h-3.5 w-3.5" />}
        </button>
      )}
    </div>
  );
}

// ── RepoLanding ──────────────────────────────────────────────────────────────

interface RepoLandingProps {
  repos: BackendRepo[];
  onSelectRepo: (repoName: string) => void;
  onAnalyzeComplete: (repoName: string) => void;
  onRemoveRepo?: (repoName: string) => Promise<void> | void;
}

export const RepoLanding = ({
  repos,
  onSelectRepo,
  onAnalyzeComplete,
  onRemoveRepo,
}: RepoLandingProps) => {
  const [removingRepo, setRemovingRepo] = useState<string | null>(null);
  const [removeError, setRemoveError] = useState<string | null>(null);

  const handleRemoveRepo = async (repoName: string) => {
    if (!onRemoveRepo || removingRepo) return;
    setRemovingRepo(repoName);
    setRemoveError(null);

    try {
      await onRemoveRepo(repoName);
    } catch (err) {
      setRemoveError(err instanceof Error ? err.message : `Failed to remove ${repoName}`);
    } finally {
      setRemovingRepo(null);
    }
  };

  return (
    <div className="press-panel press-ruled relative animate-fade-in overflow-hidden p-8">
      <div className="relative mb-6">
        <div className="text-center">
          <h1 className="press-title text-4xl leading-snug">AVmatrix</h1>
          <h2 className="press-title mt-6 text-2xl leading-snug">Open Repository</h2>
          <p className="press-reading mx-auto mt-2 text-center text-text-secondary">
            Choose an indexed local repository.
          </p>
        </div>
      </div>

      <div className="relative mb-5 space-y-2">
        {repos.map((repo) => (
          <RepoCard
            key={repo.name}
            repo={repo}
            onClick={() => onSelectRepo(repo.name)}
            onRemove={onRemoveRepo ? () => void handleRemoveRepo(repo.name) : undefined}
            isRemoving={removingRepo === repo.name}
          />
        ))}
      </div>

      {removeError && (
        <p className="mb-5 rounded-md border border-error/40 bg-error/10 px-3 py-2 text-center font-mono text-xs text-error">
          {removeError}
        </p>
      )}

      <div className="mb-5 flex items-center gap-3">
        <div className="h-px flex-1 bg-border-subtle" />
        <span className="press-eyebrow text-text-muted">
          Analyze Another Repository
        </span>
        <div className="h-px flex-1 bg-border-subtle" />
      </div>

      <div className="relative">
        <RepoAnalyzer variant="onboarding" onComplete={onAnalyzeComplete} />
      </div>

      <p className="mt-5 text-center font-mono text-[11px] leading-relaxed text-text-secondary">
        Local only. No repository data leaves this machine.
      </p>
    </div>
  );
};
