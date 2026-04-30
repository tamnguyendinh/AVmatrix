import { memo, type MutableRefObject } from 'react';
import { User, Loader2, AlertTriangle, ArrowDown } from '@/lib/lucide-icons';
import type { ChatMessage } from '../../core/llm/types.local-runtime';
import { ToolCallCard } from '../ToolCallCard';
import { MarkdownRenderer } from '../MarkdownRenderer';

const CHAT_SUGGESTIONS = [
  'Explain the project architecture',
  'What does this project do?',
  'Show me the most important files',
  'Find all API handlers',
];

interface ChatTranscriptProps {
  chatMessages: ChatMessage[];
  isChatLoading: boolean;
  isAgentReady: boolean;
  isAgentInitializing: boolean;
  agentError: string | null;
  requiresAnalyze: boolean;
  scrollContainerRef: MutableRefObject<HTMLDivElement | null>;
  messagesContainerRef: MutableRefObject<HTMLDivElement | null>;
  isAtBottom: boolean;
  scrollToBottom: () => void;
  onSuggestionSelect: (suggestion: string) => void;
  onLinkClick: (href: string) => void;
  onRequestAnalyze: () => void;
}

export const ChatTranscript = memo(function ChatTranscript({
  chatMessages,
  isChatLoading,
  isAgentReady,
  isAgentInitializing,
  agentError,
  requiresAnalyze,
  scrollContainerRef,
  messagesContainerRef,
  isAtBottom,
  scrollToBottom,
  onSuggestionSelect,
  onLinkClick,
  onRequestAnalyze,
}: ChatTranscriptProps) {
  return (
    <>
      {isAgentInitializing && (
        <div className="flex items-center justify-end gap-2 border-b border-border-subtle bg-base px-4 py-2">
          <span className="press-badge flex items-center gap-1 border-border-default bg-surface px-2 py-1 text-[11px] tracking-normal text-text-secondary normal-case">
            <Loader2 className="h-3 w-3 animate-spin" /> Connecting
          </span>
        </div>
      )}

      {agentError && (
        <div className="flex items-center gap-2 border-b border-error bg-surface px-4 py-3 text-sm text-error">
          <AlertTriangle className="h-4 w-4" />
          <span>{agentError}</span>
        </div>
      )}

      <div ref={scrollContainerRef} className="scrollbar-thin flex-1 overflow-y-auto p-4">
        {chatMessages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center px-4 text-center">
            <p className="press-eyebrow mb-2">AI assistant</p>
            <h3 className="press-title mb-2 text-2xl">Ask me anything</h3>
            <p className="press-reading mb-5 text-center text-text-secondary">
              I can help you understand the architecture, find functions, or explain connections.
            </p>
            <div className="flex flex-wrap justify-center gap-2">
              {CHAT_SUGGESTIONS.map((suggestion) => (
                <button
                  key={suggestion}
                  onClick={() => onSuggestionSelect(suggestion)}
                  className="press-outline-button rounded-full px-3 py-1.5 text-xs text-text-secondary"
                >
                  {suggestion}
                </button>
              ))}
            </div>
          </div>
        ) : (
          <div ref={messagesContainerRef} className="flex flex-col gap-6">
            {chatMessages.map((message) => (
              <div
                key={message.id}
                className={`flex animate-fade-in ${
                  message.role === 'user' ? 'justify-end' : 'justify-start'
                }`}
              >
                {message.role === 'user' && (
                  <div className="max-w-[82%]">
                    <div className="mb-2 flex items-center justify-end gap-2">
                      <span className="press-eyebrow text-text-secondary">You</span>
                      <User className="h-4 w-4 text-border-strong" />
                    </div>
                    <div className="chat-message rounded-lg border-[2px] border-border-strong bg-inset px-3.5 py-2.5 text-left">
                      {message.content}
                    </div>
                  </div>
                )}

                {message.role === 'assistant' && (
                  <div className="max-w-[92%]">
                    <div className="mb-3 flex items-center gap-2">
                      <span className="press-eyebrow text-text-secondary">My AI</span>
                      {isChatLoading && message === chatMessages[chatMessages.length - 1] && (
                        <Loader2 className="h-3 w-3 animate-spin text-border-strong" />
                      )}
                    </div>
                    <div className="chat-prose pl-6">
                      {message.steps && message.steps.length > 0 ? (
                        <div className="space-y-4">
                          {message.steps.map((step, index) => (
                            <div key={step.id}>
                              {step.type === 'reasoning' && step.content && (
                                <div className="mb-3 border-l-[3px] border-border-default pl-3 text-sm text-text-secondary italic">
                                  <MarkdownRenderer
                                    content={step.content}
                                    onLinkClick={onLinkClick}
                                  />
                                </div>
                              )}
                              {step.type === 'tool_call' && step.toolCall && (
                                <div className="mb-3">
                                  <ToolCallCard toolCall={step.toolCall} defaultExpanded={false} />
                                </div>
                              )}
                              {step.type === 'content' && step.content && (
                                <MarkdownRenderer
                                  content={step.content}
                                  onLinkClick={onLinkClick}
                                  showCopyButton={index === message.steps!.length - 1}
                                />
                              )}
                            </div>
                          ))}
                        </div>
                      ) : (
                        <MarkdownRenderer
                          content={message.content}
                          onLinkClick={onLinkClick}
                          toolCalls={message.toolCalls}
                          showCopyButton={true}
                        />
                      )}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      <button
        aria-label="Scroll to bottom"
        onClick={() => scrollToBottom()}
        className={`absolute bottom-20 left-1/2 z-10 -translate-x-1/2 rounded-full border-[2px] border-border-default bg-surface px-3 py-1.5 font-mono text-xs text-text-secondary transition-all duration-200 hover:border-border-strong hover:text-text-primary ${
          !isAtBottom && chatMessages.length > 0
            ? 'translate-y-0 opacity-100'
            : 'pointer-events-none translate-y-2 opacity-0'
        }`}
      >
        <ArrowDown className="mr-1 inline h-3.5 w-3.5" />
        Scroll to bottom
      </button>

      {requiresAnalyze && !isAgentInitializing && (
        <div className="border-t border-border-subtle bg-base px-3 pt-2 text-xs text-warning">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-3.5 w-3.5" />
            <span>This repository needs analysis before chat can start.</span>
            <button
              onClick={onRequestAnalyze}
              className="press-outline-button rounded-md border-warning px-2 py-1 text-[11px] font-medium text-warning"
            >
              Analyze now
            </button>
          </div>
        </div>
      )}
    </>
  );
});
