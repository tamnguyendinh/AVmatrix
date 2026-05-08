import { act, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DropZone } from '../../src/components/DropZone';
import { RepoLanding } from '../../src/components/RepoLanding';

const fetchReposMock = vi.fn();
const startAnalyzeMock = vi.fn();
const streamAnalyzeProgressMock = vi.fn();
const connectToServerMock = vi.fn();
const deleteRepoMock = vi.fn();
const startPollingMock = vi.fn();
const stopPollingMock = vi.fn();

vi.mock('../../src/hooks/useBackend', () => ({
  useBackend: () => ({
    isConnected: true,
    isProbing: false,
    startPolling: startPollingMock,
    stopPolling: stopPollingMock,
    isPolling: false,
    backendUrl: 'http://localhost:4747',
  }),
}));

vi.mock('../../src/services/backend-client', () => ({
  fetchRepos: (...args: unknown[]) => fetchReposMock(...args),
  startAnalyze: (...args: unknown[]) => startAnalyzeMock(...args),
  streamAnalyzeProgress: (...args: unknown[]) => streamAnalyzeProgressMock(...args),
  connectToServer: (...args: unknown[]) => connectToServerMock(...args),
  deleteRepo: (...args: unknown[]) => deleteRepoMock(...args),
}));

describe('DropZone full analyze flow', () => {
  beforeEach(() => {
    fetchReposMock.mockReset();
    startAnalyzeMock.mockReset();
    streamAnalyzeProgressMock.mockReset();
    connectToServerMock.mockReset();
    deleteRepoMock.mockReset();
    startPollingMock.mockReset();
    stopPollingMock.mockReset();

    fetchReposMock.mockResolvedValue([
      {
        name: 'AVmatrix',
        path: 'F:\\AVmatrix-main',
        indexedAt: new Date().toISOString(),
        stats: { files: 10, nodes: 20 },
      },
    ]);
    startAnalyzeMock.mockResolvedValue({ jobId: 'job-1', status: 'queued' });
    connectToServerMock.mockResolvedValue({
      nodes: [],
      relationships: [],
      repoInfo: {
        name: 'AVmatrix',
        path: 'F:\\AVmatrix-main',
        indexedAt: new Date().toISOString(),
      },
    });
  });

  it('wires repo card clicks through RepoLanding', async () => {
    const onSelectRepo = vi.fn();
    render(
      <RepoLanding
        repos={[
          {
            name: 'AVmatrix',
            path: 'F:\\AVmatrix-main',
            indexedAt: new Date().toISOString(),
            stats: { files: 10, nodes: 20 },
          },
        ]}
        onSelectRepo={onSelectRepo}
        onAnalyzeComplete={vi.fn()}
      />,
    );

    screen.getByTestId('landing-repo-card').click();

    expect(onSelectRepo).toHaveBeenCalledWith('AVmatrix');
  });

  it('runs full analyze before loading graph when a repo card is clicked', async () => {
    let completeAnalyze: ((data: { repoName?: string }) => void) | undefined;
    streamAnalyzeProgressMock.mockImplementation((_jobId, onProgress, onComplete) => {
      completeAnalyze = onComplete;
      onProgress({ phase: 'parsing', percent: 30, message: 'Parsing code' });
      return new AbortController();
    });

    render(<DropZone onServerConnect={vi.fn()} />);

    await waitFor(() => expect(fetchReposMock).toHaveBeenCalled(), { timeout: 3000 });
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 400));
    });
    const repoCard = screen.getByTestId('landing-repo-card');
    act(() => {
      repoCard.click();
    });

    await waitFor(() => {
      expect(startAnalyzeMock).toHaveBeenCalledWith({ path: 'F:\\AVmatrix-main' });
    });
    expect(connectToServerMock).not.toHaveBeenCalled();

    act(() => {
      completeAnalyze?.({ repoName: 'AVmatrix' });
    });

    await waitFor(() => {
      expect(connectToServerMock).toHaveBeenCalledWith(
        'http://localhost:4747',
        expect.any(Function),
        expect.any(AbortSignal),
        'AVmatrix',
      );
    });
  });
});
