import { useState, useRef, useEffect } from 'react';
import { Loader2, Check } from '@/lib/lucide-icons';
import {
  connectToServer,
  deleteRepo,
  fetchRepos,
  type ConnectResult,
  type BackendRepo,
} from '../services/backend-client';
import { useBackend } from '../hooks/useBackend';
import { OnboardingGuide } from './OnboardingGuide';
import { AnalyzeOnboarding } from './AnalyzeOnboarding';
import { RepoLanding } from './RepoLanding';
import { EncouragementLine } from './EncouragementLine';

interface DropZoneProps {
  onServerConnect?: (result: ConnectResult, serverUrl?: string) => void | Promise<void>;
}

// ── Crossfade wrapper ───────────────────────────────────────────────────────
// Captures the outgoing children during fade-out, then swaps to the new children on fade-in.

function Crossfade({ activeKey, children }: { activeKey: string; children: React.ReactNode }) {
  const [displayedKey, setDisplayedKey] = useState(activeKey);
  const [isTransitioning, setIsTransitioning] = useState(false);
  const snapshotRef = useRef<React.ReactNode>(children);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Keep snapshot up to date when NOT transitioning
  if (!isTransitioning && activeKey === displayedKey) {
    snapshotRef.current = children;
  }

  useEffect(() => {
    if (activeKey !== displayedKey) {
      setIsTransitioning(true);
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
      timeoutRef.current = setTimeout(() => {
        snapshotRef.current = null; // clear snapshot — new children will render
        setDisplayedKey(activeKey);
        setIsTransitioning(false);
      }, 300);
    }
    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
    };
  }, [activeKey, displayedKey]);

  return (
    <div
      className="transition-[opacity,transform] duration-300 ease-out"
      style={{
        opacity: isTransitioning ? 0 : 1,
        transform: isTransitioning ? 'scale(0.97) translateY(8px)' : 'scale(1) translateY(0)',
      }}
    >
      {isTransitioning ? snapshotRef.current : children}
    </div>
  );
}

// ── Phase cards ─────────────────────────────────────────────────────────────

function SuccessCard() {
  return (
    <div
      className="press-panel-strong press-ruled relative overflow-hidden p-8"
      role="status"
      aria-live="polite"
    >
      <div className="relative">
        <div className="mx-auto mb-5 flex h-16 w-16 items-center justify-center rounded-xl border-[3px] border-border-strong bg-inset">
          <Check className="h-8 w-8 text-success" />
        </div>

        <p className="press-eyebrow mb-2 text-center">Local bridge detected</p>
        <h2 className="press-title mb-2 text-center text-2xl">Server Connected</h2>
        <p className="mx-auto text-center font-reading leading-relaxed text-base text-text-secondary">
          Preparing your code knowledge graph...
        </p>

        <div className="mt-6 flex items-center justify-center gap-2">
          <div className="flex gap-1">
            {[0, 1, 2].map((i) => (
              <div
                key={i}
                className="h-1.5 w-1.5 animate-pulse rounded-full bg-border-strong"
                style={{ animationDelay: `${i * 200}ms` }}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function LoadingCard({ message }: { message: string }) {
  return (
    <div className="space-y-4">
      <div className="press-panel relative overflow-hidden p-8" role="status" aria-live="polite">
        <div className="relative">
          <div className="mx-auto mb-5 flex h-16 w-16 items-center justify-center rounded-xl border-[3px] border-border-default bg-inset">
            <Loader2 className="h-8 w-8 animate-spin text-border-strong" />
          </div>

          <p className="press-eyebrow mb-2 text-center">Connecting</p>
          <h2 className="press-title mb-2 text-center text-2xl">{message || 'Connecting...'}</h2>
          <p className="mx-auto text-center font-reading leading-relaxed text-base text-text-secondary">
            This may take a moment for large repositories
          </p>
        </div>
      </div>
      <EncouragementLine />
    </div>
  );
}

// ── DropZone ─────────────────────────────────────────────────────────────────

export const DropZone = ({ onServerConnect }: DropZoneProps) => {
  const [error, setError] = useState<string | null>(null);

  // Backend polling for server detection
  const {
    isConnected,
    isProbing,
    startPolling,
    stopPolling,
    isPolling,
    backendUrl: detectedBackendUrl,
  } = useBackend();
  const [initialProbeComplete, setInitialProbeComplete] = useState(false);
  const autoConnectRan = useRef(false);
  const autoConnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Connection state
  // 'analyze'  = server up but zero repos indexed — show local-path analyze input
  // 'landing'  = server up with indexed repos — show repo picker + analyze
  const [phase, setPhase] = useState<'onboarding' | 'analyze' | 'landing' | 'success' | 'loading'>(
    'onboarding',
  );
  const [loadingMessage, setLoadingMessage] = useState('');
  const abortControllerRef = useRef<AbortController | null>(null);
  const [detectedRepos, setDetectedRepos] = useState<BackendRepo[]>([]);

  // Auto-connect to the detected server — fetch repo list and show the
  // appropriate screen (landing with repo cards, or analyze for zero repos).
  const handleAutoConnect = async () => {
    setPhase('loading');
    setLoadingMessage('Connecting...');
    setError(null);

    try {
      const repos = await fetchRepos();
      if (repos.length === 0) {
        setPhase('analyze');
        autoConnectRan.current = false;
        return;
      }

      // Show landing screen so the user can choose which repo to explore
      setDetectedRepos(repos);
      setPhase('landing');
    } catch (err) {
      if ((err as Error).name === 'AbortError') return;
      const message = err instanceof Error ? err.message : 'Failed to connect';
      setError(message);
      setPhase('onboarding');
    }
  };

  const handleAutoConnectRef = useRef(handleAutoConnect);
  handleAutoConnectRef.current = handleAutoConnect;

  // Shared handler: connect to a specific repo by name (used by both repo
  // card selection on the landing screen and post-analysis completion).
  const connectToRepo = (repoName: string) => {
    autoConnectRan.current = true;
    setPhase('loading');
    setLoadingMessage('Loading graph...');
    setError(null);

    (async () => {
      const abortController = new AbortController();
      abortControllerRef.current = abortController;
      try {
        const result = await connectToServer(
          detectedBackendUrl,
          (p, downloaded, total) => {
            if (p === 'validating') {
              setLoadingMessage('Validating server...');
            } else if (p === 'downloading') {
              const pct = total ? Math.round((downloaded / total) * 100) : null;
              const mb = (downloaded / (1024 * 1024)).toFixed(1);
              setLoadingMessage(pct ? `Loading graph... ${pct}%` : `Loading graph... ${mb} MB`);
            } else if (p === 'extracting') {
              setLoadingMessage('Processing graph...');
            }
          },
          abortController.signal,
          repoName,
        );
        if (onServerConnect) {
          await onServerConnect(result, detectedBackendUrl);
        }
      } catch (err) {
        if ((err as Error).name === 'AbortError') return;
        setError(err instanceof Error ? err.message : 'Failed to load graph');
        setPhase(detectedRepos.length > 0 ? 'landing' : 'analyze');
      } finally {
        abortControllerRef.current = null;
      }
    })();
  };

  const removeRepoFromLanding = async (repoName: string) => {
    setError(null);
    await deleteRepo(repoName);

    const repos = await fetchRepos();
    setDetectedRepos(repos);
    if (repos.length === 0) {
      setPhase('analyze');
      autoConnectRan.current = false;
    }
  };

  // Track when the initial probe finishes
  useEffect(() => {
    if (!isProbing && !initialProbeComplete) {
      setInitialProbeComplete(true);
    }
  }, [isProbing, initialProbeComplete]);

  // Start polling once after initial probe fails
  useEffect(() => {
    if (initialProbeComplete && !isConnected && !isPolling && !autoConnectRan.current) {
      startPolling();
    }
  }, [initialProbeComplete, isConnected, isPolling, startPolling]);

  // Auto-connect when server is detected
  useEffect(() => {
    if (isConnected && !autoConnectRan.current) {
      autoConnectRan.current = true;
      stopPolling();
      setPhase('success');
      autoConnectTimerRef.current = setTimeout(() => {
        autoConnectTimerRef.current = null;
        handleAutoConnectRef.current();
      }, 1200); // hold success state long enough to register
    }
    // Server went away — reset to onboarding (or analyze if we were on analyze)
    if (!isConnected && autoConnectRan.current && !isProbing) {
      autoConnectRan.current = false;
      if (autoConnectTimerRef.current !== null) {
        clearTimeout(autoConnectTimerRef.current);
        autoConnectTimerRef.current = null;
      }
      setPhase('onboarding');
      setError(null);
    }
    // NOTE: No cleanup return here. The autoConnectTimerRef must survive effect
    // re-runs (e.g. isProbing flipping false while the 1200ms window is active).
    // The unmount cleanup effect below is the sole owner of timer cancellation.
  }, [isConnected, isProbing, stopPolling]);

  // Cleanup timers on unmount
  useEffect(() => {
    return () => {
      if (autoConnectTimerRef.current !== null) clearTimeout(autoConnectTimerRef.current);
      abortControllerRef.current?.abort();
    };
  }, []);

  // Don't render until initial probe completes
  const displayPhase = !initialProbeComplete ? null : phase;

  return (
    <div className="press-shell press-ruled flex min-h-screen items-center justify-center p-8">
      <div className="relative w-full max-w-lg">
        {error && (
          <div className="mb-4 animate-fade-in rounded-xl border-[3px] border-error bg-surface p-3 text-center text-sm text-error">
            {error}
          </div>
        )}

        {/* Crossfade between phases */}
        {displayPhase && (
          <Crossfade activeKey={displayPhase}>
            {displayPhase === 'onboarding' && <OnboardingGuide isPolling={isPolling} />}
            {displayPhase === 'analyze' && <AnalyzeOnboarding onComplete={connectToRepo} />}
            {displayPhase === 'landing' && (
              <RepoLanding
                repos={detectedRepos}
                onSelectRepo={connectToRepo}
                onAnalyzeComplete={connectToRepo}
                onRemoveRepo={removeRepoFromLanding}
              />
            )}
            {displayPhase === 'success' && <SuccessCard />}
            {displayPhase === 'loading' && <LoadingCard message={loadingMessage} />}
          </Crossfade>
        )}
      </div>
    </div>
  );
};
