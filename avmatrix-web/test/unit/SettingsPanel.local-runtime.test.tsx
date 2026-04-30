import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SettingsPanel } from '../../src/components/SettingsPanel.local-runtime';

const fetchSessionStatusMock = vi.fn();

vi.mock('../../src/core/llm/session-client', () => ({
  SessionClientError: class SessionClientError extends Error {
    constructor(
      message: string,
      public readonly status: number,
      public readonly code: string,
    ) {
      super(message);
      this.name = 'SessionClientError';
    }
  },
  fetchSessionStatus: (...args: unknown[]) => fetchSessionStatusMock(...args),
}));

describe('SettingsPanel.local-runtime', () => {
  beforeEach(() => {
    fetchSessionStatusMock.mockReset();
    sessionStorage.clear();
    localStorage.clear();
    fetchSessionStatusMock.mockResolvedValue({
      provider: 'codex',
      availability: 'ready',
      available: true,
      authenticated: true,
      executablePath: 'bin/codex',
      version: 'test-version',
      runtimeEnvironment: 'wsl2',
      executionMode: 'bypass',
      supportsSse: true,
      supportsCancel: true,
      supportsMcp: true,
      repo: {
        repoName: 'avmatrix',
        repoPath: 'repos/avmatrix',
        state: 'indexed',
        resolvedRepoName: 'avmatrix',
        resolvedRepoPath: 'repos/avmatrix',
      },
    });
  });

  it('renders local runtime UI without API key fields', async () => {
    render(<SettingsPanel isOpen={true} onClose={() => {}} repoName="avmatrix" />);

    expect(await screen.findByText('AI Runtime')).toBeInTheDocument();
    expect(screen.getAllByText('Codex Account').length).toBeGreaterThan(0);
    expect(screen.getByText('Claude Code')).toBeInTheDocument();
    expect(screen.queryByText('API Key')).not.toBeInTheDocument();

    await waitFor(() =>
      expect(fetchSessionStatusMock).toHaveBeenCalledWith({ repoName: 'avmatrix' }),
    );
    expect(screen.getByText('Signed in')).toBeInTheDocument();
    expect(screen.getByText('Indexed')).toBeInTheDocument();
  });

  it('re-checks local runtime status when the button is clicked', async () => {
    render(<SettingsPanel isOpen={true} onClose={() => {}} repoName="avmatrix" />);

    await screen.findByText('AI Runtime');
    await waitFor(() => expect(fetchSessionStatusMock).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByTitle('Check connection'));

    await waitFor(() => expect(fetchSessionStatusMock).toHaveBeenCalledTimes(2));
  });

  it('closes the runtime panel', async () => {
    const onClose = vi.fn();

    render(<SettingsPanel isOpen={true} onClose={onClose} repoName="avmatrix" />);

    await screen.findByText('AI Runtime');

    fireEvent.click(screen.getByText('Close'));

    expect(onClose).toHaveBeenCalled();
  });
});
