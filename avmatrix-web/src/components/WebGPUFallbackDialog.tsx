import { useState, useEffect } from 'react';
import { X, Snail, Rocket, SkipForward } from '@/lib/lucide-icons';

interface WebGPUFallbackDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onUseCPU: () => void;
  onSkip: () => void;
  nodeCount: number;
}

/**
 * Fun dialog shown when WebGPU isn't available
 * Lets user choose: CPU fallback (slow) or skip embeddings
 */
export const WebGPUFallbackDialog = ({
  isOpen,
  onClose,
  onUseCPU,
  onSkip,
  nodeCount,
}: WebGPUFallbackDialogProps) => {
  const [isAnimating, setIsAnimating] = useState(true);
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    if (isOpen) {
      // Trigger animation after mount
      requestAnimationFrame(() => setIsVisible(true));
    } else {
      setIsVisible(false);
    }
  }, [isOpen]);

  if (!isOpen) return null;

  // Estimate time based on node count (rough: ~50ms per node on CPU)
  const estimatedMinutes = Math.ceil((nodeCount * 50) / 60000);
  const isSmallCodebase = nodeCount < 200;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className={`absolute inset-0 bg-black/60 backdrop-blur-sm transition-opacity duration-200 ${isVisible ? 'opacity-100' : 'opacity-0'}`}
        onClick={onClose}
      />

      {/* Dialog */}
      <div
        className={`press-panel relative mx-4 w-full max-w-md overflow-hidden shadow-[var(--shadow-dropdown)] transition-all duration-200 ${isVisible ? 'scale-100 opacity-100' : 'scale-95 opacity-0'}`}
      >
        <div className="relative border-b-[3px] border-border-default bg-base px-6 py-5">
          <button
            onClick={onClose}
            className="press-ghost-button absolute top-4 right-4 p-1 text-text-muted"
          >
            <X className="h-5 w-5" />
          </button>

          <div className="flex items-center gap-4">
            <div
              className={`text-5xl ${isAnimating ? 'animate-bounce' : ''}`}
              onAnimationEnd={() => setIsAnimating(false)}
              onClick={() => setIsAnimating(true)}
            >
              🤔
            </div>
            <div>
              <p className="press-eyebrow">Semantic search fallback</p>
              <h2 className="press-title text-2xl">WebGPU said "nope"</h2>
              <p className="mt-0.5 font-reading text-sm text-text-secondary">
                Your browser doesn't support GPU acceleration
              </p>
            </div>
          </div>
        </div>

        <div className="space-y-4 px-6 py-5">
          <p className="press-reading text-text-secondary">
            Couldn't create embeddings with WebGPU, so semantic search (Graph RAG) won't be as
            smart. The graph still works fine though!
          </p>

          <div className="press-panel rounded-lg p-4">
            <p className="font-reading text-sm text-text-secondary">
              <span className="font-medium text-text-primary">Your options:</span>
            </p>
            <ul className="mt-2 space-y-1.5 font-reading text-sm text-text-secondary">
              <li className="flex items-start gap-2">
                <Snail className="mt-0.5 h-4 w-4 flex-shrink-0 text-warning" />
                <span>
                  <strong className="text-text-secondary">Use CPU</strong> — Works but{' '}
                  {isSmallCodebase ? 'a bit' : 'way'} slower
                  {nodeCount > 0 && (
                    <span className="text-text-muted">
                      {' '}
                      (~{estimatedMinutes} min for {nodeCount} nodes)
                    </span>
                  )}
                </span>
              </li>
              <li className="flex items-start gap-2">
                <SkipForward className="mt-0.5 h-4 w-4 flex-shrink-0 text-info" />
                <span>
                  <strong className="text-text-secondary">Skip it</strong> — Graph works, just no AI
                  semantic search
                </span>
              </li>
            </ul>
          </div>

          {isSmallCodebase && (
            <p className="flex items-center gap-1.5 rounded-lg bg-node-function/10 px-3 py-2 text-xs text-node-function">
              <Rocket className="h-3.5 w-3.5" />
              Small codebase detected! CPU should be fine.
            </p>
          )}

          <p className="font-reading text-xs text-text-secondary">
            💡 Tip: Try Chrome or Edge for WebGPU support
          </p>
        </div>

        <div className="flex gap-3 border-t-[3px] border-border-default bg-base px-6 py-4">
          <button
            onClick={onSkip}
            className="press-outline-button flex flex-1 items-center justify-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium text-text-secondary"
          >
            <SkipForward className="h-4 w-4" />
            Skip Embeddings
          </button>
          <button
            onClick={onUseCPU}
            className={`flex flex-1 items-center justify-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium transition-all ${
              isSmallCodebase
                ? 'press-filled-button'
                : 'press-outline-button border-warning text-warning hover:bg-warning/10'
            }`}
          >
            <Snail className="h-4 w-4" />
            Use CPU {isSmallCodebase ? '(Recommended)' : '(Slow)'}
          </button>
        </div>
      </div>
    </div>
  );
};
