import { memo, useCallback, useState } from 'react';
import { useAutoScroll } from '../hooks/useAutoScroll';
import { useChatRuntime } from '../hooks/chat-runtime/ChatRuntimeContext';
import { isLocalRuntimeConfigured } from '../core/llm/settings-service-local-runtime';
import { ChatTranscript } from './right-panel/ChatTranscript';
import { ChatComposer, type ComposerPrefill } from './right-panel/ChatComposer';

interface ChatPanelProps {
  onRequestAnalyze: () => void;
}

export const ChatPanel = memo(function ChatPanel({ onRequestAnalyze }: ChatPanelProps) {
  const {
    chatMessages,
    isChatLoading,
    isAgentReady,
    isAgentInitializing,
    agentError,
    sendChatMessage,
    stopChatResponse,
    clearChat,
    handleTranscriptLinkClick,
  } = useChatRuntime();
  const requiresAnalyze = agentError?.toLowerCase().includes('indexed') ?? false;
  const [composerPrefill, setComposerPrefill] = useState<ComposerPrefill | null>(null);
  const { scrollContainerRef, messagesContainerRef, isAtBottom, scrollToBottom } = useAutoScroll(
    chatMessages,
    isChatLoading,
  );

  const handleSuggestionSelect = useCallback((suggestion: string) => {
    setComposerPrefill({ id: Date.now(), text: suggestion });
  }, []);

  return (
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
        onLinkClick={handleTranscriptLinkClick}
        onRequestAnalyze={onRequestAnalyze}
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
  );
});
