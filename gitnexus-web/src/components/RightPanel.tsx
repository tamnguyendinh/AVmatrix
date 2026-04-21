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
    <aside className="relative z-30 flex w-[40%] max-w-[600px] min-w-[400px] flex-shrink-0 animate-slide-in flex-col border-l border-border-subtle bg-deep">
      {/* Header with Tabs */}
      <div className="flex items-center justify-between border-b border-border-subtle bg-surface px-4 py-2">
        <div className="flex items-center gap-1">
          {/* Chat Tab */}
          <button
            onClick={() => setRightPanelTab('chat')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              rightPanelTab === 'chat'
                ? 'bg-accent/15 text-accent'
                : 'text-text-muted hover:bg-hover hover:text-text-primary'
            }`}
          >
            <Sparkles className="h-3.5 w-3.5" />
            <span>Nexus AI</span>
          </button>

          {/* Processes Tab */}
          <button
            onClick={() => setRightPanelTab('processes')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              rightPanelTab === 'processes'
                ? 'bg-accent/15 text-accent'
                : 'text-text-muted hover:bg-hover hover:text-text-primary'
            }`}
          >
            <GitBranch className="h-3.5 w-3.5" />
            <span>Processes</span>
            <span className="rounded-full bg-gradient-to-r from-violet-500 to-fuchsia-500 px-1.5 py-0.5 text-[10px] font-semibold text-white">
              NEW
            </span>
          </button>
        </div>

        {/* Close button */}
        <button
          onClick={onClose}
          className="rounded p-1.5 text-text-muted transition-colors hover:bg-hover hover:text-text-primary"
          title="Close Panel"
        >
          <PanelRightClose className="h-4 w-4" />
        </button>
      </div>

      {/* Processes Tab */}
      {rightPanelTab === 'processes' && (
        <div className="flex flex-1 flex-col overflow-hidden">
          <ProcessesPanel />
        </div>
      )}

      {/* Chat Content - only show when chat tab is active */}
      {rightPanelTab === 'chat' && (
        <ChatPanel onRequestAnalyze={onRequestAnalyze} />
      )}
    </aside>
  );
};
