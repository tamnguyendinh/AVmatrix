import { memo, useCallback, useEffect, useRef, useState } from 'react';
import { Send, Square } from '@/lib/lucide-icons';

export interface ComposerPrefill {
  id: number;
  text: string;
}

interface ChatComposerProps {
  isChatLoading: boolean;
  isAgentInitializing: boolean;
  onSendMessage: (message: string) => Promise<void>;
  onStopResponse: () => void;
  onClearChat: () => void;
  prefill: ComposerPrefill | null;
}

export const ChatComposer = memo(function ChatComposer({
  isChatLoading,
  isAgentInitializing,
  onSendMessage,
  onStopResponse,
  onClearChat,
  prefill,
}: ChatComposerProps) {
  const [chatInput, setChatInput] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const adjustTextareaHeight = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    textarea.style.height = 'auto';
    const maxHeight = 160;
    const newHeight = Math.min(textarea.scrollHeight, maxHeight);
    textarea.style.height = `${newHeight}px`;
    textarea.style.overflowY = textarea.scrollHeight > maxHeight ? 'auto' : 'hidden';
  }, []);

  useEffect(() => {
    adjustTextareaHeight();
  }, [chatInput, adjustTextareaHeight]);

  useEffect(() => {
    if (!prefill) return;
    setChatInput(prefill.text);
  }, [prefill]);

  const resetTextarea = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;
    textarea.style.height = '36px';
    textarea.style.overflowY = 'hidden';
  }, []);

  const handleSendMessage = useCallback(async () => {
    if (!chatInput.trim()) return;
    const text = chatInput.trim();
    setChatInput('');
    resetTextarea();
    await onSendMessage(text);
  }, [chatInput, onSendMessage, resetTextarea]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        void handleSendMessage();
      }
    },
    [handleSendMessage],
  );

  return (
    <div className="border-t border-border-subtle bg-base p-3">
      <div className="press-panel flex items-end gap-2 px-3 py-2 transition-all focus-within:border-border-strong">
        <textarea
          ref={textareaRef}
          value={chatInput}
          onChange={(e) => setChatInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Ask about the codebase..."
          rows={1}
          className="chat-input scrollbar-thin min-h-[36px] flex-1 resize-none border-none bg-transparent text-sm text-text-primary outline-none placeholder:text-text-muted"
          style={{ height: '36px', overflowY: 'hidden' }}
        />
        <button
          onClick={onClearChat}
          className="press-ghost-button px-2 py-1 font-mono text-xs text-text-secondary"
          title="Clear chat"
        >
          Clear
        </button>
        {isChatLoading ? (
          <button
            onClick={onStopResponse}
            className="press-outline-button flex h-9 w-9 items-center justify-center border-error bg-surface text-error transition-all hover:bg-error/10"
            title="Stop response"
          >
            <Square className="h-3.5 w-3.5 fill-current" />
          </button>
        ) : (
          <button
            onClick={() => void handleSendMessage()}
            disabled={!chatInput.trim() || isAgentInitializing}
            className="press-filled-button flex h-9 w-9 items-center justify-center disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Send className="h-3.5 w-3.5" />
          </button>
        )}
      </div>
    </div>
  );
});
