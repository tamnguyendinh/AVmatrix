import { useState, useRef, useEffect } from 'react';
import { Check, Copy, Terminal, Server, Zap, Sparkles } from '@/lib/lucide-icons';
import { REQUIRED_NODE_VERSION } from '../config/ui-constants';

// ── Design constants ─────────────────────────────────────────────────────────

const isDev = import.meta.env.DEV;

// ── Copy-to-clipboard button ─────────────────────────────────────────────────

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard API requires secure context; localhost qualifies
    }
  };

  return (
    <button
      onClick={handleCopy}
      aria-label={copied ? 'Copied!' : 'Copy to clipboard'}
      className={`press-ghost-button shrink-0 cursor-pointer px-2 py-1 focus-visible:outline-none ${
        copied
          ? 'border-border-default bg-surface text-success'
          : 'text-text-muted'
      } `}
    >
      {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
    </button>
  );
}

// ── Faux terminal window ─────────────────────────────────────────────────────

function TerminalWindow({
  command,
  label,
  isActive = false,
}: {
  command: string;
  label: string;
  isActive?: boolean;
}) {
  return (
    <div
      className={`press-panel overflow-hidden transition-all duration-300 ${isActive ? 'border-border-strong' : ''} `}
    >
      <div className="flex items-center gap-2 border-b border-border-subtle bg-base px-4 py-2.5">
        <div className="flex gap-1.5">
          <div className="h-2.5 w-2.5 rounded-full bg-border-strong" />
          <div className="h-2.5 w-2.5 rounded-full bg-border-default" />
          <div className="h-2.5 w-2.5 rounded-full bg-border-subtle" />
        </div>
        <span className="press-eyebrow flex-1 text-center">{label}</span>
        <CopyButton text={command} />
      </div>
      <div className="flex items-center gap-3 bg-inset px-4 py-3.5 font-mono text-sm">
        <span className="text-border-strong select-none" aria-hidden="true">
          $
        </span>
        <code className="flex-1 overflow-x-auto tracking-wide whitespace-nowrap text-text-primary">
          {command}
        </code>
      </div>
    </div>
  );
}

// ── Step indicator ───────────────────────────────────────────────────────────

type StepState = 'waiting' | 'active' | 'done';

function StepDot({ state, number }: { state: StepState; number: number }) {
  if (state === 'done') {
    return (
      <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-emerald-500/50 bg-emerald-500/20">
        <Check className="h-3 w-3 text-success" />
      </div>
    );
  }
  if (state === 'active') {
      return (
        <div className="relative flex h-6 w-6 shrink-0 items-center justify-center">
        <div className="absolute inset-0 animate-ping rounded-full border border-border-default/40" />
        <div className="flex h-6 w-6 items-center justify-center rounded-full border border-border-strong bg-inset">
          <span className="text-[10px] leading-none font-semibold text-border-strong">{number}</span>
        </div>
      </div>
    );
  }
  return (
    <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-border-subtle bg-surface">
      <span className="text-[10px] leading-none font-semibold text-text-muted">{number}</span>
    </div>
  );
}

function StepRow({
  state,
  number,
  title,
  description,
  children,
}: {
  state: StepState;
  number: number;
  title: string;
  description?: string;
  children?: React.ReactNode;
}) {
  const isVisible = state !== 'waiting';

  return (
    <div
      className={`transition-all duration-300 ${state === 'waiting' ? 'opacity-40' : 'opacity-100'} `}
    >
      <div className="flex items-start gap-3">
        <StepDot state={state} number={number} />
        <div className="min-w-0 flex-1 pt-0.5">
          <div className="flex items-center gap-2">
            <span
              className={`text-sm font-medium transition-colors duration-200 ${
                state === 'done'
                  ? 'text-success'
                  : state === 'active'
                    ? 'text-text-primary'
                    : 'text-text-muted'
              }`}
            >
              {title}
            </span>
            {state === 'done' && (
                <span className="animate-fade-in font-mono text-[10px] tracking-wider text-success uppercase">
                  done
                </span>
            )}
          </div>
          {description && (
            <p className="mt-0.5 text-xs leading-relaxed text-text-muted">{description}</p>
          )}
          {isVisible && children && <div className="mt-3 animate-slide-up">{children}</div>}
        </div>
      </div>
    </div>
  );
}

// ── Polling status bar ────────────────────────────────────────────────────────

function PollingBar() {
  return (
    <div
      className="press-panel flex animate-fade-in items-center gap-3 px-4 py-3"
      aria-live="polite"
      role="status"
    >
      <div className="relative shrink-0">
        <Zap className="h-4 w-4 text-border-strong" />
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="h-5 w-5 animate-pulse rounded-full border border-border-strong/40" />
        </div>
      </div>

      <div className="min-w-0 flex-1">
        <p className="text-xs font-medium text-text-secondary">
          Listening for local bridge
          <span className="ml-0.5 inline-flex text-text-muted">
            <span className="animate-pulse">...</span>
          </span>
        </p>
        <p className="mt-0.5 text-[11px] text-text-muted">Will auto-connect when detected</p>
      </div>
    </div>
  );
}

// ── OnboardingGuide ───────────────────────────────────────────────────────────

interface OnboardingGuideProps {
  isPolling?: boolean;
}

export const OnboardingGuide = ({ isPolling }: OnboardingGuideProps) => {
  const primary = isDev ? 'cd avmatrix && npm run serve' : 'avmatrix serve';
  const termLabel = 'Start local bridge';

  // Step states: step 1 = copy command, step 2 = run/wait, step 3 = auto-connect
  // Once polling starts the user has presumably run the command — mark step 1 done.
  const step1State: StepState = isPolling ? 'done' : 'active';
  const step2State: StepState = isPolling ? 'active' : 'waiting';
  const step3State: StepState = 'waiting';

  return (
    <div className="press-panel press-ruled relative animate-fade-in overflow-hidden p-8">
      <div className="relative mb-6">
        <div className="text-center">
          <p className="press-eyebrow mb-3 inline-flex items-center gap-1.5">
            <Sparkles className="h-3.5 w-3.5 text-border-strong" />
            <span>Vol. XII · Local bridge</span>
          </p>
          <h2 className="press-title text-3xl leading-snug">
            Start AVmatrix locally
          </h2>
          <p className="press-reading mx-auto mt-3 text-center text-text-secondary">
            {isDev
              ? 'Start the local bridge in a separate terminal. The browser stays local and auto-connects when it is ready.'
              : 'One local command is all it takes. The browser connects automatically.'}
          </p>
        </div>
      </div>

      <div className="relative space-y-5">
        <div
          className="pointer-events-none absolute top-6 bottom-6 left-[11px] w-px bg-border-subtle"
          aria-hidden="true"
        />

        <StepRow
          state={step1State}
          number={1}
          title="Copy the command"
          description={isPolling ? undefined : 'Click the icon in the terminal to copy.'}
        >
          <TerminalWindow command={primary} label={termLabel} isActive={step1State === 'active'} />

          {!isDev && (
            <>
              <div className="my-3 flex items-center gap-3">
                <div className="h-px flex-1 bg-border-subtle" />
                <span className="text-[11px] tracking-widest text-text-muted uppercase">
                  or install globally
                </span>
                <div className="h-px flex-1 bg-border-subtle" />
              </div>
              <TerminalWindow
                command="npm install -g avmatrix && avmatrix serve"
                label="Global install"
                isActive={false}
              />
            </>
          )}
        </StepRow>

        {/* Step 2 — Run and wait */}
        <StepRow
          state={step2State}
          number={2}
          title={isPolling ? 'Waiting for local bridge to start' : 'Paste and run in your terminal'}
          description={
            isPolling ? undefined : 'Open a terminal at the project root, paste, and hit Enter.'
          }
        >
          {isPolling && <PollingBar />}
        </StepRow>

        {/* Step 3 — Auto-connect */}
        <StepRow
          state={step3State}
          number={3}
          title="Auto-connects and opens the graph"
          description="No refresh needed — the page detects the server automatically."
        />
      </div>

      <div className="mt-6 flex items-center justify-center gap-1.5 border-t border-border-subtle pt-5 text-xs text-text-secondary">
        <Server className="h-3 w-3 shrink-0 text-border-strong" />
        <span>
          Requires{' '}
          <a
            href="https://nodejs.org"
            target="_blank"
            rel="noopener noreferrer"
            className="text-border-strong transition-colors hover:text-accent-dim hover:underline"
          >
            Node.js {REQUIRED_NODE_VERSION}+
          </a>
        </span>
        <span className="mx-1 text-border-default">·</span>
        <Terminal className="h-3 w-3 shrink-0" />
        <span>Port 4747</span>
      </div>
    </div>
  );
};
