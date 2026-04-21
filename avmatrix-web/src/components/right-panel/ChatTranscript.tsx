import { memo, type MutableRefObject } from 'react';
import {
  Sparkles,
  User,
  Loader2,
  AlertTriangle,
  ArrowDown,
} from '@/lib/lucide-icons';
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
  isLocalRuntimeConfigured: boolean;
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
  isLocalRuntimeConfigured,
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
      <div className="flex items-center gap-2.5 border-b border-border-subtle bg-elevated/50 px-4 py-3">
        <div className="ml-auto flex items-center gap-2">
          {!isAgentReady && (
            <span className="rounded-full border border-amber-500/30 bg-amber-500/15 px-2 py-1 text-[11px] text-amber-300">
              Session offline
            </span>
          )}
          {isAgentInitializing && (
            <span className="flex items-center gap-1 rounded-full border border-border-subtle bg-surface px-2 py-1 text-[11px] text-text-muted">
              <Loader2 className="h-3 w-3 animate-spin" /> Connecting
            </span>
          )}
        </div>
      </div>

      {agentError && (
        <div className="flex items-center gap-2 border-b border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100">
          <AlertTriangle className="h-4 w-4" />
          <span>{agentError}</span>
        </div>
      )}

      <div ref={scrollContainerRef} className="scrollbar-thin flex-1 overflow-y-auto p-4">
        {chatMessages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center px-4 text-center">
            <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-xl bg-gradient-to-br from-accent to-node-interface text-2xl shadow-glow">
              🧠
            </div>
            <h3 className="mb-2 text-base font-medium">Ask me anything</h3>
            <p className="mb-5 text-sm leading-relaxed text-text-secondary">
              I can help you understand the architecture, find functions, or explain
              connections.
            </p>
            <div className="flex flex-wrap justify-center gap-2">
              {CHAT_SUGGESTIONS.map((suggestion) => (
                <button
                  key={suggestion}
                  onClick={() => onSuggestionSelect(suggestion)}
                  className="rounded-full border border-border-subtle bg-elevated px-3 py-1.5 text-xs text-text-secondary transition-colors hover:border-accent hover:text-text-primary"
                >
                  {suggestion}
                </button>
              ))}
            </div>
          </div>
        ) : (
          <div ref={messagesContainerRef} className="flex flex-col gap-6">
            {chatMessages.map((message) => (
              <div key={message.id} className="animate-fade-in">
                {message.role === 'user' && (
                  <div className="mb-4">
                    <div className="mb-2 flex items-center gap-2">
                      <User className="h-4 w-4 text-text-muted" />
                      <span className="text-xs font-medium tracking-wide text-text-muted uppercase">
                        You
                      </span>
                    </div>
                    <div className="pl-6 text-sm text-text-primary">{message.content}</div>
                  </div>
                )}

                {message.role === 'assistant' && (
                  <div>
                    <div className="mb-3 flex items-center gap-2">
                      <Sparkles className="h-4 w-4 text-accent" />
                      <span className="text-xs font-medium tracking-wide text-text-muted uppercase">
                        Nexus AI
                      </span>
                      {isChatLoading && message === chatMessages[chatMessages.length - 1] && (
                        <Loader2 className="h-3 w-3 animate-spin text-accent" />
                      )}
                    </div>
                    <div className="chat-prose pl-6">
                      {message.steps && message.steps.length > 0 ? (
                        <div className="space-y-4">
                          {message.steps.map((step, index) => (
                            <div key={step.id}>
                              {step.type === 'reasoning' && step.content && (
                                <div className="mb-3 border-l-2 border-text-muted/30 pl-3 text-sm text-text-secondary italic">
                                  <MarkdownRenderer
                                    content={step.content}
                                    onLinkClick={onLinkClick}
                                  />
                                </div>
                              )}
                              {step.type === 'tool_call' && step.toolCall && (
                                <div className="mb-3">
                                  <ToolCallCard
                                    toolCall={step.toolCall}
                                    defaultExpanded={false}
                                  />
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
        className={`absolute bottom-20 left-1/2 z-10 -translate-x-1/2 rounded-full border border-border-subtle bg-elevated px-3 py-1.5 text-xs text-text-secondary shadow-lg transition-all duration-200 hover:border-accent hover:text-accent ${
          !isAtBottom && chatMessages.length > 0
            ? 'translate-y-0 opacity-100'
            : 'pointer-events-none translate-y-2 opacity-0'
        }`}
      >
        <ArrowDown className="mr-1 inline h-3.5 w-3.5" />
        Scroll to bottom
      </button>

      {!isAgentReady && !isAgentInitializing && (
        <div className="border-t border-border-subtle bg-surface px-3 pt-2 text-xs text-amber-200">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-3.5 w-3.5" />
            <span>
              {requiresAnalyze
                ? 'This repository needs analysis before chat can start.'
                : isLocalRuntimeConfigured
                  ? 'Local session runtime is not ready yet.'
                  : 'Local session runtime is unavailable.'}
            </span>
            {requiresAnalyze && (
              <button
                onClick={onRequestAnalyze}
                className="rounded-md border border-amber-500/30 bg-amber-500/10 px-2 py-1 text-[11px] font-medium text-amber-100 transition-colors hover:bg-amber-500/20"
              >
                Analyze now
              </button>
            )}
          </div>
        </div>
      )}
    </>
  );
});
