import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppStateProvider } from '../../src/hooks/useAppState.local-runtime';

const mockChatPanel = vi.fn(({ onRequestAnalyze }: { onRequestAnalyze: () => void }) => (
  <button onClick={onRequestAnalyze}>Chat Panel</button>
));

vi.mock('../../src/components/ProcessesPanel', () => ({
  ProcessesPanel: () => <div>Processes</div>,
}));

vi.mock('../../src/components/ChatPanel', () => ({
  ChatPanel: (props: { onRequestAnalyze: () => void }) => mockChatPanel(props),
}));

import { RightPanel } from '../../src/components/RightPanel';

describe('RightPanel.local-runtime', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the chat shell and forwards analyze requests to ChatPanel', () => {
    const onRequestAnalyze = vi.fn();

    render(
      <AppStateProvider>
        <RightPanel isOpen={true} onClose={vi.fn()} onRequestAnalyze={onRequestAnalyze} />
      </AppStateProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Chat Panel' }));
    expect(onRequestAnalyze).toHaveBeenCalledTimes(1);
  });

  it('switches between chat and processes tabs without changing the shell UI', () => {
    render(
      <AppStateProvider>
        <RightPanel isOpen={true} onClose={vi.fn()} onRequestAnalyze={vi.fn()} />
      </AppStateProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: /Processes/i }));
    expect(screen.getAllByText('Processes')).toHaveLength(2);
  });
});
