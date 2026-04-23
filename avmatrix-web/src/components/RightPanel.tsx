import { Sparkles, PanelRightClose, GitBranch } from '@/lib/lucide-icons';
import { useAppState } from '../hooks/useAppState.local-runtime';
import { ProcessesPanel } from './ProcessesPanel';
import { ChatPanel } from './ChatPanel';

interface RightPanelProps {
  isOpen: boolean;
  onClose: () => void;
  onRequestAnalyze: () => void;
}

export const RightPanel = ({ isOpen, onClose, onRequestAnalyze }: RightPanelProps) => {
  const { rightPanelTab, setRightPanelTab } = useAppState();

  if (!isOpen) return null;

  return (
    <aside className="relative z-30 flex w-[40%] max-w-[600px] min-w-[400px] flex-shrink-0 animate-slide-in flex-col border-l-[3px] border-border-default bg-surface">
      <div className="flex items-center justify-between border-b-[3px] border-border-default bg-base px-4 py-3">
        <div className="flex items-center gap-1">
          <button
            onClick={() => setRightPanelTab('chat')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-2 font-mono text-sm font-medium transition-colors ${
              rightPanelTab === 'chat'
                ? 'border-[3px] border-border-strong bg-surface text-text-primary'
                : 'border-[3px] border-transparent text-text-muted hover:bg-surface hover:text-text-primary'
            }`}
          >
            <Sparkles className="h-3.5 w-3.5" />
            <span>Desk Chat</span>
          </button>

          <button
            onClick={() => setRightPanelTab('processes')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-2 font-mono text-sm font-medium transition-colors ${
              rightPanelTab === 'processes'
                ? 'border-[3px] border-border-strong bg-surface text-text-primary'
                : 'border-[3px] border-transparent text-text-muted hover:bg-surface hover:text-text-primary'
            }`}
          >
            <GitBranch className="h-3.5 w-3.5" />
            <span>Processes</span>
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

      {rightPanelTab === 'processes' && (
        <div className="flex flex-1 flex-col overflow-hidden">
          <ProcessesPanel />
        </div>
      )}

      {rightPanelTab === 'chat' && (
        <ChatPanel onRequestAnalyze={onRequestAnalyze} />
      )}
    </aside>
  );
};
