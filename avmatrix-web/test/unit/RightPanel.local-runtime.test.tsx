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

import { RightPanelResizable } from '../../src/components/RightPanel.resizable';

describe('RightPanelResizable.local-runtime', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      writable: true,
      value: 1600,
    });
  });

  it('renders the chat shell and forwards analyze requests to ChatPanel', () => {
    const onRequestAnalyze = vi.fn();

    render(
      <AppStateProvider>
        <RightPanelResizable isOpen={true} onClose={vi.fn()} onRequestAnalyze={onRequestAnalyze} />
      </AppStateProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Chat Panel' }));
    expect(onRequestAnalyze).toHaveBeenCalledTimes(1);
  });

  it('switches between chat and processes tabs without changing the shell UI', () => {
    render(
      <AppStateProvider>
        <RightPanelResizable isOpen={true} onClose={vi.fn()} onRequestAnalyze={vi.fn()} />
      </AppStateProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: /Processes/i }));
    expect(screen.getAllByText('Processes')).toHaveLength(2);
  });

  it('resizes the panel from the drag handle and persists width', () => {
    const { container } = render(
      <AppStateProvider>
        <RightPanelResizable isOpen={true} onClose={vi.fn()} onRequestAnalyze={vi.fn()} />
      </AppStateProvider>,
    );

    const panel = container.querySelector('aside');
    expect(panel).not.toBeNull();
    expect(panel?.style.width).toBe('520px');

    const resizeHandle = screen.getByTitle('Drag to resize panel');
    fireEvent.mouseDown(resizeHandle, { clientX: 900 });
    fireEvent.mouseMove(window, { clientX: 980 });
    fireEvent.mouseUp(window);

    expect(panel?.style.width).toBe('440px');
    expect(window.localStorage.getItem('avmatrix.rightPanelWidth')).toBe('440');
  });
});
