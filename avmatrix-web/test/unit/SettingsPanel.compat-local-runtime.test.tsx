import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SettingsPanel } from '../../src/components/SettingsPanel';

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

describe('SettingsPanel compatibility wrapper', () => {
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

  it('preserves the legacy component contract while rendering the local runtime UI', async () => {
    render(<SettingsPanel isOpen={true} onClose={() => {}} repoName="avmatrix" />);

    expect(await screen.findByText('AI Runtime')).toBeInTheDocument();
    expect(screen.getAllByText('Codex Account').length).toBeGreaterThan(0);
    expect(screen.queryByText('Configure your LLM provider')).not.toBeInTheDocument();
    expect(screen.queryByText('API Key')).not.toBeInTheDocument();

    await waitFor(() =>
      expect(fetchSessionStatusMock).toHaveBeenCalledWith({ repoName: 'avmatrix' }),
    );
  });
});
