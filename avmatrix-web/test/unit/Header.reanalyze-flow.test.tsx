import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const startAnalyzeMock = vi.fn();
const streamAnalyzeProgressMock = vi.fn();

vi.mock('../../src/components/RepoAnalyzer', () => ({
  RepoAnalyzer: () => <div data-testid="repo-analyzer">repo-analyzer</div>,
}));
vi.mock('../../src/components/EmbeddingStatus', () => ({
  EmbeddingStatus: () => <div data-testid="embedding-status">embedding-status</div>,
}));
vi.mock('../../src/hooks/useAppState.local-runtime', () => ({
  useAppState: () => ({
    projectName: 'Website',
    graph: null,
    openChatPanel: vi.fn(),
    isRightPanelOpen: false,
    rightPanelTab: 'chat',
    setSettingsPanelOpen: vi.fn(),
    setHelpDialogBoxOpen: vi.fn(),
  }),
}));
vi.mock('../../src/services/backend-client', () => ({
  deleteRepo: vi.fn(),
  fetchRepos: vi.fn(),
  startAnalyze: (...args: unknown[]) => startAnalyzeMock(...args),
  streamAnalyzeProgress: (...args: unknown[]) => streamAnalyzeProgressMock(...args),
}));

import { Header } from '../../src/components/Header';

describe('Header re-analyze flow', () => {
  beforeEach(() => {
    startAnalyzeMock.mockReset();
    streamAnalyzeProgressMock.mockReset();
    startAnalyzeMock.mockResolvedValue({ jobId: 'job-1', status: 'queued' });
  });

  it('starts analyze by repo path and reloads the completed graph', async () => {
    const onAnalyzeComplete = vi.fn();
    let completeAnalyze: (() => void) | undefined;
    streamAnalyzeProgressMock.mockImplementation((_jobId, _onProgress, onComplete) => {
      completeAnalyze = onComplete;
      return new AbortController();
    });

    render(
      <Header
        availableRepos={[
          {
            name: 'Website',
            path: 'F:\\Website',
            indexedAt: new Date().toISOString(),
          },
        ]}
        onAnalyzeComplete={onAnalyzeComplete}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: /Website/i }));
    fireEvent.click(screen.getByTitle('Re-analyze Website'));

    await waitFor(() => {
      expect(startAnalyzeMock).toHaveBeenCalledWith({ path: 'F:\\Website' });
    });

    act(() => {
      completeAnalyze?.();
    });

    expect(onAnalyzeComplete).toHaveBeenCalledWith('Website');
  });
});
