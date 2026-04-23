/**
 * RepoAnalyzer
 *
 * Local-only input:
 *   - "local" → Paste an absolute local folder path for analysis
 */

import { useState, useRef, useEffect, useId } from 'react';
import {
  FolderOpen,
  Loader2,
  Check,
  ArrowRight,
  AlertCircle,
  Sparkles,
} from '@/lib/lucide-icons';
import {
  startAnalyze,
  cancelAnalyze,
  streamAnalyzeProgress,
  type JobProgress,
} from '../services/backend-client';
import { AnalyzeProgress } from './AnalyzeProgress';

// ── Helpers ──────────────────────────────────────────────────────────────────

const IS_WINDOWS = navigator.userAgent.toLowerCase().includes('win');

function isLikelyAbsoluteLocalPath(value: string): boolean {
  const trimmed = value.trim();
  if (!trimmed) return false;
  if (trimmed.startsWith('\\\\') || trimmed.startsWith('//')) return false;
  if (IS_WINDOWS) {
    return /^[a-zA-Z]:[\\/]/.test(trimmed);
  }
  return trimmed.startsWith('/');
}

// ── Analyze button ───────────────────────────────────────────────────────────

function AnalyzeButton({
  canSubmit,
  isLoading,
  onClick,
  variant,
}: {
  canSubmit: boolean;
  isLoading: boolean;
  onClick: () => void;
  variant: 'onboarding' | 'sheet';
}) {
  const sizeClass =
    variant === 'onboarding' ? 'w-full px-5 py-3.5 text-sm' : 'w-full px-4 py-3 text-sm';
  return (
    <button
      onClick={onClick}
      disabled={!canSubmit || isLoading}
      className={`press-filled-button ${sizeClass} flex items-center justify-center gap-2.5 ${
        canSubmit && !isLoading
          ? 'cursor-pointer'
          : 'cursor-not-allowed border-border-subtle bg-inset text-text-muted'
      } `}
    >
      {isLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
      <span>{isLoading ? 'Starting analysis...' : 'Analyze Repository'}</span>
      {canSubmit && !isLoading && <ArrowRight className="h-3.5 w-3.5" />}
    </button>
  );
}

// ── Done state ───────────────────────────────────────────────────────────────

function DoneState({ repoName }: { repoName: string }) {
  return (
    <div
      className="flex animate-fade-in flex-col items-center gap-3 py-4"
      role="status"
      aria-live="polite"
    >
      <div className="flex h-12 w-12 items-center justify-center rounded-xl border-[3px] border-border-strong bg-inset">
        <Check className="h-6 w-6 text-success" />
      </div>
      <div className="text-center">
        <p className="press-eyebrow text-success">Analysis complete</p>
        <p className="mt-0.5 font-mono text-xs text-text-muted">{repoName}</p>
      </div>
      <p className="text-xs font-reading text-text-secondary">Loading graph...</p>
    </div>
  );
}

// ── RepoAnalyzer ─────────────────────────────────────────────────────────────

type InternalPhase = 'input' | 'starting' | 'analyzing' | 'done' | 'error';

export interface RepoAnalyzerProps {
  variant: 'onboarding' | 'sheet';
  onComplete: (repoName: string) => void;
  onCancel?: () => void;
}

export const RepoAnalyzer = ({ variant, onComplete, onCancel }: RepoAnalyzerProps) => {
  const inputId = useId();
  const folderInputRef = useRef<HTMLInputElement>(null);
  const [localPath, setLocalPath] = useState('');
  const [phase, setPhase] = useState<InternalPhase>('input');
  const [validationError, setValidationError] = useState<string | null>(null);
  const [progress, setProgress] = useState<JobProgress>({
    phase: 'queued',
    percent: 0,
    message: 'Queued',
  });
  const [completedRepoName, setCompletedRepoName] = useState('');

  const jobIdRef = useRef<string | null>(null);
  const sseControllerRef = useRef<AbortController | null>(null);
  const completeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      sseControllerRef.current?.abort();
      if (completeTimerRef.current) clearTimeout(completeTimerRef.current);
    };
  }, []);

  // Use the browser's native directory picker (webkitdirectory doesn't give paths,
  // so we use a text input + a "Browse" button that opens a standard file input
  // to let users pick files from the folder — the path is typed manually since
  // browsers don't expose absolute paths for security reasons).
  // For local paths, the user types or pastes the absolute path.

  const canSubmit =
    isLikelyAbsoluteLocalPath(localPath) && (phase === 'input' || phase === 'error');

  const handleAnalyze = async () => {
    if (!isLikelyAbsoluteLocalPath(localPath)) {
      setValidationError('Please enter an absolute local folder path.');
      return;
    }

    setValidationError(null);
    setPhase('starting');

    try {
      const request = { path: localPath.trim() };
      const { jobId } = await startAnalyze(request);
      jobIdRef.current = jobId;
      setPhase('analyzing');

      const nameSource = localPath.trim();
      const controller = streamAnalyzeProgress(
        jobId,
        (p) => setProgress(p),
        (data) => {
          const name =
            data.repoName ?? nameSource.split(/[/\\]/).filter(Boolean).at(-1) ?? 'repository';
          setCompletedRepoName(name);
          setPhase('done');
          sseControllerRef.current = null;
          completeTimerRef.current = setTimeout(() => {
            completeTimerRef.current = null;
            onComplete(name);
          }, 1200);
        },
        (errMsg) => {
          setValidationError(errMsg || 'Analysis failed. Check server logs.');
          setPhase('error');
        },
      );
      sseControllerRef.current = controller;
    } catch (err) {
      setValidationError(err instanceof Error ? err.message : 'Failed to start analysis');
      setPhase('error');
    }
  };

  const handleCancel = async () => {
    sseControllerRef.current?.abort();
    sseControllerRef.current = null;
    if (jobIdRef.current) {
      try {
        await cancelAnalyze(jobIdRef.current);
      } catch {}
      jobIdRef.current = null;
    }
    setPhase('input');
    setProgress({ phase: 'queued', percent: 0, message: 'Queued' });
  };

  const isLoading = phase === 'starting';
  const showInput = phase !== 'analyzing' && phase !== 'done';
  const isWindows = IS_WINDOWS;

  return (
    <div className="space-y-4">
      {/* Local folder input */}
      {showInput && (
        <div className="space-y-2">
          <label
            htmlFor={`${inputId}-local`}
            className="press-eyebrow block"
          >
            Local Folder Path
          </label>
          <div
            className={`press-inset flex items-center gap-3 px-4 py-3.5 transition-all duration-200 ${
              validationError && phase === 'error'
                ? 'border-error'
                : isLikelyAbsoluteLocalPath(localPath)
                  ? 'border-border-strong shadow-focus'
                  : 'focus-within:border-border-strong'
            } `}
          >
            <FolderOpen className="h-4 w-4 shrink-0 text-text-muted" />
            <input
              id={`${inputId}-local`}
              type="text"
              value={localPath}
              onChange={(e) => {
                setLocalPath(e.target.value);
                if (validationError) setValidationError(null);
              }}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && canSubmit && !isLoading) {
                  e.preventDefault();
                  handleAnalyze();
                }
              }}
              disabled={isLoading}
              placeholder={isWindows ? 'C:\\Users\\you\\project' : '/home/you/project'}
              autoComplete="off"
              spellCheck={false}
              className="flex-1 border-none bg-transparent font-mono text-sm text-text-primary outline-none placeholder:text-text-muted disabled:opacity-50"
            />
            {isLikelyAbsoluteLocalPath(localPath) && (
              <Check className="h-3.5 w-3.5 shrink-0 text-success" />
            )}
          </div>
          {/* Native folder picker + Browse button — below the input */}
          <input
            ref={folderInputRef}
            type="file"
            // @ts-expect-error -- webkitdirectory is non-standard but widely supported
            webkitdirectory=""
            className="hidden"
            onChange={(e) => {
              const files = e.target.files;
              if (files && files.length > 0) {
                setValidationError(
                  'Browsers do not expose absolute folder paths here. Paste the full local path manually.',
                );
              }
              e.target.value = '';
            }}
          />
          <button
            type="button"
            onClick={() => folderInputRef.current?.click()}
            disabled={isLoading}
            className="press-outline-button flex w-full cursor-pointer items-center justify-center gap-2 px-3 py-2 text-xs font-medium text-text-secondary disabled:opacity-50"
          >
            <FolderOpen className="h-3.5 w-3.5" />
            Folder picker hint
          </button>
          <p className="text-xs text-text-muted">
            Paste an absolute local path. Browser folder pickers cannot reveal the full path, so the
            picker only helps confirm the folder name.
          </p>
        </div>
      )}

      {/* Error message */}
      {(phase === 'error' || (phase === 'input' && validationError)) && validationError && (
        <p className="flex animate-fade-in items-center gap-1.5 text-xs text-error">
          <AlertCircle className="h-3 w-3 shrink-0" />
          {validationError}
        </p>
      )}

      {/* Live progress */}
      {phase === 'analyzing' && (
        <div className="animate-slide-up">
          <AnalyzeProgress progress={progress} onCancel={handleCancel} />
        </div>
      )}

      {/* Done */}
      {phase === 'done' && <DoneState repoName={completedRepoName} />}

      {/* CTA button */}
      {(phase === 'input' || phase === 'starting') && (
        <AnalyzeButton
          canSubmit={canSubmit}
          isLoading={isLoading}
          onClick={handleAnalyze}
          variant={variant}
        />
      )}

      {/* Error retry */}
      {phase === 'error' && (
        <div className="flex items-center gap-3">
          <button
            onClick={() => {
              setValidationError(null);
              setPhase('input');
            }}
            className="press-outline-button flex-1 cursor-pointer px-4 py-2.5 text-sm text-text-secondary"
          >
            Try again
          </button>
          {onCancel && (
            <button
              onClick={onCancel}
              className="press-ghost-button cursor-pointer px-4 py-2.5 text-sm text-text-secondary"
            >
              Dismiss
            </button>
          )}
        </div>
      )}

      {/* Dismiss for sheet variant while analyzing */}
      {phase === 'analyzing' && variant === 'sheet' && onCancel && (
        <button
          onClick={onCancel}
          className="press-ghost-button w-full cursor-pointer py-1 text-xs text-text-secondary"
        >
          Hide (analysis continues in background)
        </button>
      )}
    </div>
  );
};
