import { useCallback, useEffect, useState } from 'react';
import { Sparkles, PanelRightClose, GitBranch } from '@/lib/lucide-icons';
import { useAppState } from '../hooks/useAppState.local-runtime';
import { ProcessesPanel } from './ProcessesPanel';
import { ChatPanel } from './ChatPanel';

interface RightPanelResizableProps {
  isOpen: boolean;
  onClose: () => void;
  onRequestAnalyze: () => void;
}

const RIGHT_PANEL_STORAGE_KEY = 'avmatrix.rightPanelWidth';
const RIGHT_PANEL_MIN_WIDTH = 360;
const RIGHT_PANEL_DEFAULT_WIDTH = 520;
const RIGHT_PANEL_MAX_WIDTH = 760;

const clampPanelWidth = (width: number, viewportWidth: number): number => {
  const viewportMax = Math.max(
    RIGHT_PANEL_MIN_WIDTH + 40,
    Math.floor(viewportWidth * 0.58),
  );
  const maxWidth = Math.min(RIGHT_PANEL_MAX_WIDTH, viewportMax);
  return Math.max(RIGHT_PANEL_MIN_WIDTH, Math.min(width, maxWidth));
};

const loadStoredWidth = (): number => {
  if (typeof window === 'undefined') return RIGHT_PANEL_DEFAULT_WIDTH;
  try {
    const saved = window.localStorage.getItem(RIGHT_PANEL_STORAGE_KEY);
    const parsed = saved ? parseInt(saved, 10) : NaN;
    if (!Number.isFinite(parsed)) {
      return clampPanelWidth(RIGHT_PANEL_DEFAULT_WIDTH, window.innerWidth);
    }
    return clampPanelWidth(parsed, window.innerWidth);
  } catch {
    return clampPanelWidth(RIGHT_PANEL_DEFAULT_WIDTH, window.innerWidth);
  }
};

export const RightPanelResizable = ({
  isOpen,
  onClose,
  onRequestAnalyze,
}: RightPanelResizableProps) => {
  const { rightPanelTab, setRightPanelTab } = useAppState();
  const [panelWidth, setPanelWidth] = useState(loadStoredWidth);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      window.localStorage.setItem(RIGHT_PANEL_STORAGE_KEY, String(panelWidth));
    } catch {
      // ignore localStorage write failures
    }
  }, [panelWidth]);

  useEffect(() => {
    if (typeof window === 'undefined') return;

    const handleResize = () => {
      setPanelWidth((prev) => clampPanelWidth(prev, window.innerWidth));
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const startResize = useCallback(
    (event: React.MouseEvent<HTMLDivElement>) => {
      event.preventDefault();
      event.stopPropagation();

      const startX = event.clientX;
      const startWidth = panelWidth;
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';

      const onMove = (moveEvent: MouseEvent) => {
        const deltaX = moveEvent.clientX - startX;
        const nextWidth = clampPanelWidth(
          startWidth - deltaX,
          window.innerWidth,
        );
        setPanelWidth(nextWidth);
      };

      const onUp = () => {
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
        window.removeEventListener('mousemove', onMove);
        window.removeEventListener('mouseup', onUp);
      };

      window.addEventListener('mousemove', onMove);
      window.addEventListener('mouseup', onUp);
    },
    [panelWidth],
  );

  if (!isOpen) return null;

  return (
    <aside
      className="relative z-30 my-3 mr-3 ml-0 flex min-h-0 self-stretch animate-slide-in flex-col overflow-hidden rounded-[18px] border-[3px] border-border-default bg-surface shadow-[var(--shadow-dropdown)]"
      style={{ width: panelWidth }}
    >
      <div
        onMouseDown={startResize}
        className="absolute top-0 left-0 z-10 h-full w-3 cursor-col-resize bg-transparent"
        title="Drag to resize panel"
      >
        <div className="absolute top-1/2 left-0 h-18 w-[3px] -translate-y-1/2 rounded-full bg-border-subtle transition-colors hover:bg-border-strong" />
      </div>

      <div className="flex items-center justify-between border-b-[3px] border-border-default bg-base px-4 py-3">
        <div className="flex min-w-0 items-center gap-1">
          <button
            onClick={() => setRightPanelTab('chat')}
            className={`flex min-w-0 items-center gap-1.5 rounded-md px-3 py-2 font-mono text-sm font-medium transition-colors ${
              rightPanelTab === 'chat'
                ? 'border-[3px] border-border-strong bg-surface text-text-primary'
                : 'border-[3px] border-transparent text-text-muted hover:bg-surface hover:text-text-primary'
            }`}
          >
            <Sparkles className="h-3.5 w-3.5 shrink-0" />
            <span className="truncate">Desk Chat</span>
          </button>

          <button
            onClick={() => setRightPanelTab('processes')}
            className={`flex min-w-0 items-center gap-1.5 rounded-md px-3 py-2 font-mono text-sm font-medium transition-colors ${
              rightPanelTab === 'processes'
                ? 'border-[3px] border-border-strong bg-surface text-text-primary'
                : 'border-[3px] border-transparent text-text-muted hover:bg-surface hover:text-text-primary'
            }`}
          >
            <GitBranch className="h-3.5 w-3.5 shrink-0" />
            <span className="truncate">Processes</span>
            <span className="press-badge border-border-default bg-base px-1.5 py-0.5 text-[10px] text-text-secondary">
              NEW
            </span>
          </button>
        </div>

        <button
          onClick={onClose}
          className="press-ghost-button rounded p-1.5 text-text-muted"
          title="Close Panel"
        >
          <PanelRightClose className="h-4 w-4" />
        </button>
      </div>

      <div className="flex min-h-0 flex-1 flex-col bg-surface">
        {rightPanelTab === 'processes' && (
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
            <ProcessesPanel />
          </div>
        )}

        {rightPanelTab === 'chat' && (
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
            <ChatPanel onRequestAnalyze={onRequestAnalyze} />
          </div>
        )}
      </div>
    </aside>
  );
};
