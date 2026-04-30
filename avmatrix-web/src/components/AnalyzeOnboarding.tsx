/**
 * AnalyzeOnboarding
 *
 * The "empty state" card rendered inside DropZone's Crossfade when the server
 * is connected but zero repos are indexed. Replaces the generic error message
 * with a first-class local path analysis flow.
 *
 * Rendering context:
 *   DropZone (Crossfade, phase="analyze")
 *     └─ AnalyzeOnboarding
 *          └─ RepoAnalyzer (variant="onboarding")
 *
 * When the analysis job completes, onComplete fires with the repoName, and
 * DropZone's handleAutoConnect re-runs (now that repos > 0), transitioning
 * the app to the graph explorer.
 */

import { FolderOpen } from '@/lib/lucide-icons';
import { RepoAnalyzer } from './RepoAnalyzer';

interface AnalyzeOnboardingProps {
  /** Called when analysis finishes and the repo is ready to load. */
  onComplete: (repoName: string) => void;
}

export const AnalyzeOnboarding = ({ onComplete }: AnalyzeOnboardingProps) => {
  return (
    <div className="press-panel press-ruled relative animate-fade-in overflow-hidden p-8">
      <div className="relative mb-6">
        <div className="text-center">
          <div className="mb-3 flex justify-center">
            <span className="press-badge border-border-default bg-base text-text-primary">
              AVmatrix
            </span>
          </div>
          <p className="press-eyebrow mb-3">Edition 01 · First indexing run</p>

          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl border-[3px] border-border-strong bg-inset">
            <FolderOpen className="h-7 w-7 text-border-strong" />
          </div>

          <h2 className="press-title text-3xl leading-snug">Analyze your first repository</h2>
          <p className="press-reading mx-auto mt-3 text-center text-text-secondary">
            Paste an absolute local repository path and AVmatrix will parse the code and build a
            live knowledge graph — entirely on this machine.
          </p>
        </div>
      </div>

      <div className="relative">
        <RepoAnalyzer variant="onboarding" onComplete={onComplete} />
      </div>

      <p className="mt-5 text-center font-mono text-[11px] leading-relaxed text-text-secondary">
        Local repositories only &middot; Parsed by the AVmatrix local runtime &middot; No data
        leaves your machine
      </p>
    </div>
  );
};
