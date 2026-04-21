import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { X, Server, Brain, Check, AlertCircle, RefreshCw, Loader2 } from '@/lib/lucide-icons';
import type { SessionStatusResponse } from 'gitnexus-shared';
import {
  fetchSessionStatus,
  SessionClientError,
} from '../core/llm/session-client';
import {
  getLocalRuntimeAvailableModels,
  getLocalRuntimeProviderDisplayName,
  loadLocalRuntimeSettings,
  saveLocalRuntimeSettings,
} from '../core/llm/settings-service-local-runtime';
import type { LocalRuntimeSettings } from '../core/llm/settings-service-local-runtime';

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
    badge: 'bg-green-400',
    panel: 'border-green-500/30 bg-green-500/10 text-green-200',
    label: 'Ready',
  },
  not_installed: {
    badge: 'bg-amber-400',
    panel: 'border-amber-500/30 bg-amber-500/10 text-amber-200',
    label: 'Not installed',
  },
  not_signed_in: {
    badge: 'bg-amber-400',
    panel: 'border-amber-500/30 bg-amber-500/10 text-amber-200',
    label: 'Sign-in required',
  },
  error: {
    badge: 'bg-red-400',
    panel: 'border-red-500/30 bg-red-500/10 text-red-200',
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
  return `${status.runtimeEnvironment.toUpperCase()} · ${status.executionMode}`;
};

const DetailRow = ({ label, value }: { label: string; value: string }) => (
  <div className="rounded-lg border border-border-subtle bg-elevated px-3 py-2">
    <p className="text-[11px] tracking-wide text-text-muted uppercase">{label}</p>
    <p className="mt-1 text-sm text-text-primary">{value}</p>
  </div>
);

export const SettingsPanel = ({
  isOpen,
  onClose,
  onSettingsSaved,
  backendUrl,
  isBackendConnected,
  onBackendUrlChange,
  repoName,
}: SettingsPanelProps) => {
  const [settings, setSettings] = useState<LocalRuntimeSettings>(loadLocalRuntimeSettings);
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saved' | 'error'>('idle');
  const [sessionStatus, setSessionStatus] = useState<SessionStatusResponse | null>(null);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [isCheckingStatus, setIsCheckingStatus] = useState(false);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const codexModels = useMemo(() => getLocalRuntimeAvailableModels('codex'), []);
  const tone = availabilityTone[sessionStatus?.availability ?? 'error'];

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
    return () => {
      if (saveTimerRef.current) {
        clearTimeout(saveTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (!isOpen) return;
    setSettings(loadLocalRuntimeSettings());
    setSaveStatus('idle');
    void refreshStatus();
  }, [isOpen, refreshStatus]);

  const handleSave = () => {
    try {
      saveLocalRuntimeSettings(settings);
      setSaveStatus('saved');
      onSettingsSaved?.();
      if (saveTimerRef.current) {
        clearTimeout(saveTimerRef.current);
      }
      saveTimerRef.current = setTimeout(() => setSaveStatus('idle'), 2000);
    } catch {
      setSaveStatus('error');
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />

      <div className="relative mx-4 flex max-h-[90vh] w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-border-subtle bg-surface shadow-2xl">
        <div className="flex items-center justify-between border-b border-border-subtle bg-elevated/50 px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-accent/20">
              <Brain className="h-5 w-5 text-accent" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-text-primary">Session Settings</h2>
              <p className="text-xs text-text-muted">
                Configure your local Codex / Claude Code session runtime
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-2 text-text-muted transition-colors hover:bg-hover hover:text-text-primary"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex-1 space-y-6 overflow-y-auto p-6">
          {backendUrl !== undefined && onBackendUrlChange && (
            <div className="space-y-3">
              <label className="block text-sm font-medium text-text-secondary">Local Server</label>
              <div className="space-y-2">
                <div className="mb-2 flex items-center gap-2">
                  <Server className="h-4 w-4 text-text-muted" />
                  <span className="text-sm text-text-secondary">Backend URL</span>
                  <span
                    className={`h-2 w-2 rounded-full ${isBackendConnected ? 'bg-green-400' : 'bg-red-400'}`}
                  />
                  <span className="text-xs text-text-muted">
                    {isBackendConnected ? 'Connected' : 'Not connected'}
                  </span>
                </div>
                <input
                  type="url"
                  aria-label="Backend URL"
                  value={backendUrl}
                  onChange={(e) => onBackendUrlChange(e.target.value)}
                  placeholder="http://localhost:4747"
                  className="w-full rounded-xl border border-border-subtle bg-elevated px-4 py-3 font-mono text-sm text-text-primary transition-all outline-none placeholder:text-text-muted focus:border-accent focus:ring-2 focus:ring-accent/20"
                />
                <p className="text-xs text-text-muted">
                  Run <code className="rounded bg-elevated px-1 py-0.5">gitnexus serve</code> to
                  host the local runtime bridge on this machine.
                </p>
              </div>
            </div>
          )}

          <div className="space-y-3">
            <label className="block text-sm font-medium text-text-secondary">Session Runtime</label>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="flex items-center gap-3 rounded-xl border-2 border-accent bg-accent/10 p-4 text-text-primary">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent/20 text-lg">
                  🧠
                </div>
                <div className="min-w-0">
                  <p className="font-medium">{getLocalRuntimeProviderDisplayName('codex')}</p>
                  <p className="text-xs text-text-muted">Active in Phase 2</p>
                </div>
              </div>
              <div className="flex items-center gap-3 rounded-xl border-2 border-border-subtle bg-elevated p-4 text-text-secondary opacity-70">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-surface text-lg">
                  ✨
                </div>
                <div className="min-w-0">
                  <p className="font-medium">Claude Code</p>
                  <p className="text-xs text-text-muted">Adapter slot reserved</p>
                </div>
              </div>
            </div>
          </div>

          <div className="rounded-xl border border-accent/20 bg-accent/5 p-3 text-xs text-accent/90">
            No API keys are stored here. AVmatrix uses your local session runtime and the account
            already signed in on this machine.
          </div>

          <div className="animate-fade-in space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-sm font-semibold text-text-primary">Codex Account</h3>
                <p className="text-xs text-text-muted">
                  Local session status for the current browser + CLI workflow.
                </p>
              </div>
              <button
                type="button"
                onClick={() => void refreshStatus()}
                disabled={isCheckingStatus}
                className="rounded-xl border border-border-subtle bg-elevated px-3 py-2 text-text-secondary transition-colors hover:border-accent/50 hover:text-text-primary disabled:opacity-50"
                title="Check connection"
              >
                {isCheckingStatus ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4" />
                )}
              </button>
            </div>

            <div className={`rounded-xl border p-4 ${statusError ? 'border-red-500/30 bg-red-500/10 text-red-200' : tone.panel}`}>
              <div className="flex items-start gap-3">
                <div
                  className={`mt-1 h-2.5 w-2.5 flex-shrink-0 rounded-full ${statusError ? 'bg-red-400' : tone.badge}`}
                />
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">
                    {statusError ? 'Connection failed' : tone.label}
                  </p>
                  <p className="mt-1 text-xs leading-relaxed">
                    {statusError ||
                      sessionStatus?.message ||
                      'The local session runtime is ready for chat and tool execution.'}
                  </p>
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-text-secondary">Model</label>
              <select
                aria-label="Model"
                value={settings.codex?.model ?? 'codex-account'}
                onChange={(e) =>
                  setSettings((prev) => ({
                    ...prev,
                    activeProvider: 'codex',
                    codex: { ...prev.codex, model: e.target.value },
                  }))
                }
                className="w-full rounded-xl border border-border-subtle bg-elevated px-4 py-3 font-mono text-sm text-text-primary transition-all outline-none focus:border-accent focus:ring-2 focus:ring-accent/20"
              >
                {codexModels.map((model) => (
                  <option key={model} value={model}>
                    {model}
                  </option>
                ))}
              </select>
              <p className="text-xs text-text-muted">
                The browser stores only the local runtime preference. Authentication stays in the
                Codex CLI session.
              </p>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-text-secondary">Temperature</label>
                <input
                  type="number"
                  aria-label="Temperature"
                  min="0"
                  max="2"
                  step="0.1"
                  value={settings.codex?.temperature ?? 0}
                  onChange={(e) =>
                    setSettings((prev) => ({
                      ...prev,
                      activeProvider: 'codex',
                      codex: {
                        ...prev.codex,
                        temperature: Number(e.target.value),
                      },
                    }))
                  }
                  className="w-full rounded-xl border border-border-subtle bg-elevated px-4 py-3 font-mono text-sm text-text-primary transition-all outline-none focus:border-accent focus:ring-2 focus:ring-accent/20"
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-text-secondary">Max Tokens</label>
                <input
                  type="number"
                  aria-label="Max Tokens"
                  min="0"
                  step="1"
                  value={settings.codex?.maxTokens ?? ''}
                  onChange={(e) =>
                    setSettings((prev) => ({
                      ...prev,
                      activeProvider: 'codex',
                      codex: {
                        ...prev.codex,
                        maxTokens: e.target.value ? Number(e.target.value) : undefined,
                      },
                    }))
                  }
                  placeholder="Optional"
                  className="w-full rounded-xl border border-border-subtle bg-elevated px-4 py-3 font-mono text-sm text-text-primary transition-all outline-none placeholder:text-text-muted focus:border-accent focus:ring-2 focus:ring-accent/20"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <DetailRow
                label="Account"
                value={
                  sessionStatus
                    ? sessionStatus.authenticated
                      ? 'Signed in'
                      : 'Not signed in'
                    : 'Unknown'
                }
              />
              <DetailRow
                label="Runtime"
                value={formatRuntimeEnvironment(sessionStatus)}
              />
              <DetailRow
                label="Repository"
                value={repoStateLabel(sessionStatus?.repo)}
              />
              <DetailRow
                label="Version"
                value={sessionStatus?.version ?? 'Unknown'}
              />
            </div>

            {sessionStatus?.repo?.state === 'index_required' && (
              <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-200">
                Analyze the current repository from the repository menu before starting a chat
                session. The runtime will not auto-index from the chat path.
              </div>
            )}

            {sessionStatus?.executablePath && (
              <div className="rounded-xl border border-border-subtle bg-elevated/50 p-3">
                <p className="text-[11px] tracking-wide text-text-muted uppercase">Executable</p>
                <p className="mt-1 break-all font-mono text-xs text-text-secondary">
                  {sessionStatus.executablePath}
                </p>
              </div>
            )}
          </div>

          <div className="rounded-xl border border-border-subtle bg-elevated/50 p-4">
            <div className="flex gap-3">
              <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-green-500/20 text-green-400">
                🔒
              </div>
              <div className="text-xs leading-relaxed text-text-muted">
                <span className="font-medium text-text-secondary">Privacy:</span> AVmatrix keeps
                only lightweight UI settings in browser storage. Repository data stays on this
                machine, and no AVmatrix-hosted proxy is involved in the session path.
              </div>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-between border-t border-border-subtle bg-elevated/30 px-6 py-4">
          <div className="flex items-center gap-2 text-sm">
            {saveStatus === 'saved' && (
              <span className="flex animate-fade-in items-center gap-1.5 text-green-400">
                <Check className="h-4 w-4" />
                Settings saved
              </span>
            )}
            {saveStatus === 'error' && (
              <span className="flex animate-fade-in items-center gap-1.5 text-red-400">
                <AlertCircle className="h-4 w-4" />
                Failed to save
              </span>
            )}
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm text-text-secondary transition-colors hover:text-text-primary"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              className="rounded-lg bg-accent px-5 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-dim"
            >
              Save Settings
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
