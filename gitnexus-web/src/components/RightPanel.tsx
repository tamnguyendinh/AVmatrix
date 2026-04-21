import { useState, useCallback } from 'react';
import { Sparkles, PanelRightClose, GitBranch } from '@/lib/lucide-icons';
import { useAppState } from '../hooks/useAppState.local-runtime';
import { useAutoScroll } from '../hooks/useAutoScroll';
import { isLocalRuntimeConfigured } from '../core/llm/settings-service-local-runtime';
import { ProcessesPanel } from './ProcessesPanel';
import { ChatTranscript } from './right-panel/ChatTranscript';
import { ChatComposer, type ComposerPrefill } from './right-panel/ChatComposer';
export const RightPanel = () => {
  const {
    isRightPanelOpen,
    setRightPanelOpen,
    graph,
    addCodeReference,
    // LLM / chat state
    chatMessages,
    isChatLoading,
    agentError,
    isAgentReady,
    isAgentInitializing,
    sendChatMessage,
    stopChatResponse,
    clearChat,
    requestRepoAnalyzeDialog,
  } = useAppState();

  const requiresAnalyze = agentError?.toLowerCase().includes('indexed') ?? false;

  const [activeTab, setActiveTab] = useState<'chat' | 'processes'>('chat');
  const [composerPrefill, setComposerPrefill] = useState<ComposerPrefill | null>(null);
  // Keep streamed replies pinned unless the user intentionally scrolls away from the bottom.
  const { scrollContainerRef, messagesContainerRef, isAtBottom, scrollToBottom } = useAutoScroll(
    chatMessages,
    isChatLoading,
  );

  const resolveFilePathForUI = useCallback((_requestedPath: string): string | null => {
    return null;
  }, []);

  const findFileNodeIdForUI = useCallback(
    (filePath: string): string | undefined => {
      if (!graph) return undefined;
      const target = filePath.replace(/\\/g, '/').replace(/^\.?\//, '');
      const node = graph.nodes.find(
        (n) =>
          n.label === 'File' &&
          n.properties.filePath.replace(/\\/g, '/').replace(/^\.?\//, '') === target,
      );
      return node?.id;
    },
    [graph],
  );

  const handleGroundingClick = useCallback(
    (inner: string) => {
      const raw = inner.trim();
      if (!raw) return;

      let rawPath = raw;
      let startLine1: number | undefined;
      let endLine1: number | undefined;

      // Match line:num or line:num-num (supports both hyphen - and en dash –)
      const lineMatch = raw.match(/^(.*):(\d+)(?:[-–](\d+))?$/);
      if (lineMatch) {
        rawPath = lineMatch[1].trim();
        startLine1 = parseInt(lineMatch[2], 10);
        endLine1 = parseInt(lineMatch[3] || lineMatch[2], 10);
      }

      const resolvedPath = resolveFilePathForUI(rawPath);
      if (!resolvedPath) return;

      const nodeId = findFileNodeIdForUI(resolvedPath);

      addCodeReference({
        filePath: resolvedPath,
        startLine: startLine1 ? Math.max(0, startLine1 - 1) : undefined,
        endLine: endLine1
          ? Math.max(0, endLine1 - 1)
          : startLine1
            ? Math.max(0, startLine1 - 1)
            : undefined,
        nodeId,
        label: 'File',
        name: resolvedPath.split('/').pop() ?? resolvedPath,
        source: 'ai',
      });
    },
    [addCodeReference, findFileNodeIdForUI, resolveFilePathForUI],
  );

  // Handler for node grounding: [[Class:View]], [[Function:trigger]], etc.
  const handleNodeGroundingClick = useCallback(
    (nodeTypeAndName: string) => {
      const raw = nodeTypeAndName.trim();
      if (!raw || !graph) return;

      // Parse Type:Name format
      const match = raw.match(
        /^(Class|Function|Method|Interface|File|Folder|Variable|Enum|Type|CodeElement):(.+)$/,
      );
      if (!match) return;

      const [, nodeType, nodeName] = match;
      const trimmedName = nodeName.trim();

      // Find node in graph by type + name
      const node = graph.nodes.find(
        (n) => n.label === nodeType && n.properties.name === trimmedName,
      );

      if (!node) {
        console.warn(`Node not found: ${nodeType}:${trimmedName}`);
        return;
      }

      // 1. Highlight in graph (add to AI citation highlights)
      // Note: This requires accessing the state setter from parent context
      // For now, we'll add to code references which triggers the highlight

      // 2. Add to Code Panel (if node has file/line info)
      if (node.properties.filePath) {
        const resolvedPath = resolveFilePathForUI(node.properties.filePath);
        if (resolvedPath) {
          addCodeReference({
            filePath: resolvedPath,
            startLine: node.properties.startLine ? node.properties.startLine - 1 : undefined,
            endLine: node.properties.endLine ? node.properties.endLine - 1 : undefined,
            nodeId: node.id,
            label: node.label,
            name: node.properties.name,
            source: 'ai',
          });
        }
      }
    },
    [graph, resolveFilePathForUI, addCodeReference],
  );

  const handleLinkClick = useCallback(
    (href: string) => {
      if (href.startsWith('code-ref:')) {
        const inner = decodeURIComponent(href.slice('code-ref:'.length));
        handleGroundingClick(inner);
      } else if (href.startsWith('node-ref:')) {
        const inner = decodeURIComponent(href.slice('node-ref:'.length));
        handleNodeGroundingClick(inner);
      }
    },
    [handleGroundingClick, handleNodeGroundingClick],
  );

  const handleSuggestionSelect = useCallback((suggestion: string) => {
    setComposerPrefill({ id: Date.now(), text: suggestion });
  }, []);

  if (!isRightPanelOpen) return null;

  return (
    <aside className="relative z-30 flex w-[40%] max-w-[600px] min-w-[400px] flex-shrink-0 animate-slide-in flex-col border-l border-border-subtle bg-deep">
      {/* Header with Tabs */}
      <div className="flex items-center justify-between border-b border-border-subtle bg-surface px-4 py-2">
        <div className="flex items-center gap-1">
          {/* Chat Tab */}
          <button
            onClick={() => setActiveTab('chat')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              activeTab === 'chat'
                ? 'bg-accent/15 text-accent'
                : 'text-text-muted hover:bg-hover hover:text-text-primary'
            }`}
          >
            <Sparkles className="h-3.5 w-3.5" />
            <span>Nexus AI</span>
          </button>

          {/* Processes Tab */}
          <button
            onClick={() => setActiveTab('processes')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              activeTab === 'processes'
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
          onClick={() => setRightPanelOpen(false)}
          className="rounded p-1.5 text-text-muted transition-colors hover:bg-hover hover:text-text-primary"
          title="Close Panel"
        >
          <PanelRightClose className="h-4 w-4" />
        </button>
      </div>

      {/* Processes Tab */}
      {activeTab === 'processes' && (
        <div className="flex flex-1 flex-col overflow-hidden">
          <ProcessesPanel />
        </div>
      )}

      {/* Chat Content - only show when chat tab is active */}
      {activeTab === 'chat' && (
        <div className="relative flex flex-1 flex-col overflow-hidden">
          <ChatTranscript
            chatMessages={chatMessages}
            isChatLoading={isChatLoading}
            isAgentReady={isAgentReady}
            isAgentInitializing={isAgentInitializing}
            agentError={agentError}
            requiresAnalyze={requiresAnalyze}
            isLocalRuntimeConfigured={isLocalRuntimeConfigured()}
            scrollContainerRef={scrollContainerRef}
            messagesContainerRef={messagesContainerRef}
            isAtBottom={isAtBottom}
            scrollToBottom={scrollToBottom}
            onSuggestionSelect={handleSuggestionSelect}
            onLinkClick={handleLinkClick}
            onRequestAnalyze={requestRepoAnalyzeDialog}
          />
          <ChatComposer
            isChatLoading={isChatLoading}
            isAgentInitializing={isAgentInitializing}
            onSendMessage={sendChatMessage}
            onStopResponse={stopChatResponse}
            onClearChat={clearChat}
            prefill={composerPrefill}
          />
        </div>
      )}
    </aside>
  );
};
