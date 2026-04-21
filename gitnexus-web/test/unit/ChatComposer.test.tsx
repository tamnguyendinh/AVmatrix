import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ChatComposer } from '../../src/components/right-panel/ChatComposer';

describe('ChatComposer', () => {
  it('applies a suggestion prefill without losing the current composer UI', async () => {
    const send = vi.fn().mockResolvedValue(undefined);

    const { rerender } = render(
      <ChatComposer
        isChatLoading={false}
        isAgentInitializing={false}
        onSendMessage={send}
        onStopResponse={vi.fn()}
        onClearChat={vi.fn()}
        prefill={null}
      />,
    );

    const textarea = screen.getByPlaceholderText('Ask about the codebase...');
    expect(textarea).toHaveValue('');

    rerender(
      <ChatComposer
        isChatLoading={false}
        isAgentInitializing={false}
        onSendMessage={send}
        onStopResponse={vi.fn()}
        onClearChat={vi.fn()}
        prefill={{ id: 1, text: 'Explain the project architecture' }}
      />,
    );

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Ask about the codebase...')).toHaveValue(
        'Explain the project architecture',
      );
    });
  });

  it('sends on Enter and preserves newline behavior on Shift+Enter', async () => {
    const send = vi.fn().mockResolvedValue(undefined);

    render(
      <ChatComposer
        isChatLoading={false}
        isAgentInitializing={false}
        onSendMessage={send}
        onStopResponse={vi.fn()}
        onClearChat={vi.fn()}
        prefill={null}
      />,
    );

    const textarea = screen.getByPlaceholderText('Ask about the codebase...');
    fireEvent.change(textarea, { target: { value: 'hello' } });
    fireEvent.keyDown(textarea, { key: 'Enter', shiftKey: true });
    expect(send).not.toHaveBeenCalled();

    fireEvent.keyDown(textarea, { key: 'Enter', shiftKey: false });
    await waitFor(() => {
      expect(send).toHaveBeenCalledWith('hello');
    });
  });

  it('keeps clear and stop actions wired', () => {
    const onClearChat = vi.fn();
    const onStopResponse = vi.fn();

    const { rerender } = render(
      <ChatComposer
        isChatLoading={false}
        isAgentInitializing={false}
        onSendMessage={vi.fn().mockResolvedValue(undefined)}
        onStopResponse={onStopResponse}
        onClearChat={onClearChat}
        prefill={null}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Clear' }));
    expect(onClearChat).toHaveBeenCalledTimes(1);

    rerender(
      <ChatComposer
        isChatLoading={true}
        isAgentInitializing={false}
        onSendMessage={vi.fn().mockResolvedValue(undefined)}
        onStopResponse={onStopResponse}
        onClearChat={onClearChat}
        prefill={null}
      />,
    );

    fireEvent.click(screen.getByTitle('Stop response'));
    expect(onStopResponse).toHaveBeenCalledTimes(1);
  });
});
