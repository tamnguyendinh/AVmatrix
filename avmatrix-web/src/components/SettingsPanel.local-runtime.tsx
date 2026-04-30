import { useCallback, useEffect, useState } from 'react';
import { X, Server, RefreshCw, Loader2 } from '@/lib/lucide-icons';
import type { SessionStatusResponse } from 'avmatrix-shared';
import { fetchSessionStatus, SessionClientError } from '../core/llm/session-client';

interface SettingsPanelProps {
  isOpen: boolean;
  onClose: () => void;
  onSettingsSaved?: () => void;
  backendUrl?: string;
  isBackendConnected?: boolean;
  onBackendUrlChange?: (url: string) => void;
  repoName?: string;
}

const availabilityTone: Record<
  NonNullable<SessionStatusResponse['availability']>,
  { badge: string; panel: string; label: string }
> = {
  ready: {
    badge: 'bg-success',
    panel: 'border-success bg-base text-text-primary',
    label: 'Ready',
  },
  not_installed: {
    badge: 'bg-warning',
    panel: 'border-warning bg-base text-text-primary',
    label: 'Not installed',
  },
  not_signed_in: {
    badge: 'bg-warning',
    panel: 'border-warning bg-base text-text-primary',
    label: 'Sign-in required',
  },
  error: {
    badge: 'bg-error',
    panel: 'border-error bg-base text-text-primary',
    label: 'Unavailable',
  },
};

const repoStateLabel = (status?: SessionStatusResponse['repo']): string => {
  switch (status?.state) {
    case 'indexed':
      return 'Indexed';
    case 'index_required':
      return 'Analyze required';
    case 'not_found':
      return 'Repo not found';
    case 'invalid':
      return 'Invalid binding';
    default:
      return 'Not bound';
  }
};

const formatRuntimeEnvironment = (status?: SessionStatusResponse | null): string => {
  if (!status) return 'Unknown';
  const runtime =
    status.runtimeEnvironment === 'wsl2'
      ? 'WSL2'
      : status.runtimeEnvironment.charAt(0).toUpperCase() + status.runtimeEnvironment.slice(1);
  return `${runtime} · ${status.executionMode}`;
};

const DetailRow = ({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: 'success' | 'warning' | 'error';
}) => (
  <div className="rounded-lg border-[2px] border-border-default bg-base px-3 py-2">
    <p className="press-eyebrow text-text-secondary">{label}</p>
    <p
      className={`mt-1 font-mono text-sm font-semibold ${
        tone === 'success'
          ? 'text-success'
          : tone === 'warning'
            ? 'text-warning'
            : tone === 'error'
              ? 'text-error'
              : 'text-text-primary'
      }`}
    >
      {value}
    </p>
  </div>
);

export const SettingsPanel = ({
  isOpen,
  onClose,
  backendUrl,
  isBackendConnected,
  repoName,
}: SettingsPanelProps) => {
  const [sessionStatus, setSessionStatus] = useState<SessionStatusResponse | null>(null);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [isCheckingStatus, setIsCheckingStatus] = useState(false);

  const tone = availabilityTone[sessionStatus?.availability ?? 'error'];
  const codexConnected =
    !statusError && Boolean(sessionStatus?.available && sessionStatus.authenticated);
  const codexStatusLabel =
    isCheckingStatus && !sessionStatus
      ? 'Checking'
      : codexConnected
        ? 'Connected'
        : 'Not connected';

  const refreshStatus = useCallback(async () => {
    setIsCheckingStatus(true);
    setStatusError(null);

    try {
      const status = await fetchSessionStatus(repoName ? { repoName } : undefined);
      setSessionStatus(status);
    } catch (error) {
      const message =
        error instanceof SessionClientError || error instanceof Error
          ? error.message
          : 'Failed to reach the local session runtime';
      setStatusError(message);
      setSessionStatus(null);
    } finally {
      setIsCheckingStatus(false);
    }
  }, [repoName]);

  useEffect(() => {
    if (!isOpen) return;
    void refreshStatus();
  }, [isOpen, refreshStatus]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />

      <div className="press-panel relative mx-4 flex max-h-[90vh] w-full max-w-xl flex-col overflow-hidden shadow-[var(--shadow-dropdown)]">
        <div className="flex items-center justify-between border-b-[3px] border-border-default bg-base px-6 py-5">
          <div>
            <h2 className="press-title text-2xl">AI Runtime</h2>
            <p className="font-reading text-sm text-text-secondary">Local Codex session status</p>
          </div>
          <button onClick={onClose} className="press-ghost-button rounded-lg p-2 text-text-muted">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex-1 space-y-5 overflow-y-auto p-6">
          <p className="font-reading text-sm leading-relaxed text-text-secondary">
            AVmatrix uses the Codex CLI session already available on this machine.
          </p>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div
              className={`rounded-xl border-[3px] bg-base p-4 ${
                codexConnected ? 'border-success' : 'border-border-default'
              }`}
            >
              <p className="press-eyebrow text-text-secondary">Codex Account</p>
              <div className="mt-3 flex items-center justify-between gap-3">
                <p className="font-mono font-semibold text-base text-text-primary">Codex CLI</p>
                <span
                  className={`rounded-full border-[2px] px-3 py-1 font-mono text-xs font-semibold ${
                    codexConnected
                      ? 'border-success text-success'
                      : 'border-border-default text-text-secondary'
                  }`}
                >
                  {codexStatusLabel}
                </span>
              </div>
            </div>

            <div className="rounded-xl border-[3px] border-border-default bg-surface p-4">
              <p className="press-eyebrow text-text-secondary">Claude Code Account</p>
              <div className="mt-3 flex items-center justify-between gap-3">
                <p className="font-mono font-semibold text-base text-text-primary">Claude Code</p>
                <span className="rounded-full border-[2px] border-border-default px-3 py-1 font-mono text-xs font-semibold text-text-secondary">
                  Not connected
                </span>
              </div>
            </div>
          </div>

          <div className="animate-fade-in space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="press-title text-xl">Connection</h3>
              </div>
              <button
                type="button"
                onClick={() => void refreshStatus()}
                disabled={isCheckingStatus}
                className="press-outline-button rounded-xl px-3 py-2 text-text-secondary disabled:opacity-50"
                title="Check connection"
              >
                {isCheckingStatus ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4" />
                )}
              </button>
            </div>

            {statusError && (
              <div className="rounded-xl border-[3px] border-error bg-base p-4 text-text-primary">
                <div className="flex items-start gap-3">
                  <div className="mt-1 h-2.5 w-2.5 flex-shrink-0 rounded-full bg-error" />
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium">Connection failed</p>
                    <p className="mt-1 text-xs leading-relaxed">{statusError}</p>
                  </div>
                </div>
              </div>
            )}

            <div className="grid grid-cols-2 gap-3">
              <DetailRow
                label="Local Server"
                value={isBackendConnected ? 'Connected' : 'Not connected'}
                tone={isBackendConnected ? 'success' : 'error'}
              />
              <DetailRow
                label="Codex CLI"
                value={tone.label}
                tone={sessionStatus?.availability === 'ready' ? 'success' : 'warning'}
              />
              <DetailRow
                label="Account"
                value={
                  sessionStatus
                    ? sessionStatus.authenticated
                      ? 'Signed in'
                      : 'Not signed in'
                    : 'Unknown'
                }
                tone={sessionStatus?.authenticated ? 'success' : 'warning'}
              />
              <DetailRow
                label="Repository"
                value={repoStateLabel(sessionStatus?.repo)}
                tone={sessionStatus?.repo?.state === 'indexed' ? 'success' : 'warning'}
              />
            </div>

            {sessionStatus?.repo?.state === 'index_required' && (
              <div className="rounded-xl border-[3px] border-warning bg-surface p-3 font-reading text-xs text-warning">
                Analyze the current repository from the repository menu before starting a chat
                session. The runtime will not auto-index from the chat path.
              </div>
            )}
          </div>

          <div className="space-y-3">
            <h3 className="press-title text-xl">Runtime Details</h3>
            <div className="grid grid-cols-2 gap-3">
              <DetailRow label="Mode" value={formatRuntimeEnvironment(sessionStatus)} />
              <DetailRow label="Version" value={sessionStatus?.version ?? 'Unknown'} />
              <DetailRow label="Executable" value={sessionStatus?.executablePath ?? 'Unknown'} />
              <DetailRow label="Backend URL" value={backendUrl ?? 'http://localhost:4747'} />
            </div>
          </div>

          <div className="rounded-xl border-[3px] border-border-default bg-base p-4">
            <div className="flex gap-3">
              <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border-[2px] border-border-default bg-surface text-success">
                <Server className="h-4 w-4" />
              </div>
              <div className="font-reading text-xs leading-relaxed text-text-secondary">
                Local only. AVmatrix does not store API keys or route chat through an AVmatrix cloud
                service.
              </div>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-end border-t-[3px] border-border-default bg-base px-6 py-4">
          <button
            onClick={onClose}
            className="press-filled-button rounded-lg px-5 py-2 text-sm font-medium"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
