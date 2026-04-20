import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SettingsPanel } from '../../src/components/SettingsPanel.local-runtime';
import { loadLocalRuntimeSettings } from '../../src/core/llm/settings-service-local-runtime';

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
      version: '0.119.0',
      runtimeEnvironment: 'wsl2',
      executionMode: 'bypass',
      supportsSse: true,
      supportsCancel: true,
      supportsMcp: true,
      repo: {
        repoName: 'GitNexus',
        repoPath: 'repos/GitNexus',
        state: 'indexed',
        resolvedRepoName: 'GitNexus',
        resolvedRepoPath: 'repos/GitNexus',
      },
    });
  });

  it('renders local runtime UI without API key fields', async () => {
    render(<SettingsPanel isOpen={true} onClose={() => {}} repoName="GitNexus" />);

    expect(await screen.findByText('Session Settings')).toBeInTheDocument();
    expect(screen.getAllByText('Codex Account').length).toBeGreaterThan(0);
    expect(screen.getByText('Claude Code')).toBeInTheDocument();
    expect(screen.queryByText('API Key')).not.toBeInTheDocument();

    await waitFor(() =>
      expect(fetchSessionStatusMock).toHaveBeenCalledWith({ repoName: 'GitNexus' }),
    );
    expect(screen.getByText('Signed in')).toBeInTheDocument();
    expect(screen.getByText('Indexed')).toBeInTheDocument();
  });

  it('re-checks local runtime status when the button is clicked', async () => {
    render(<SettingsPanel isOpen={true} onClose={() => {}} repoName="GitNexus" />);

    await screen.findByText('Session Settings');
    await waitFor(() => expect(fetchSessionStatusMock).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByTitle('Check connection'));

    await waitFor(() => expect(fetchSessionStatusMock).toHaveBeenCalledTimes(2));
  });

  it('saves codex runtime preferences and notifies the parent', async () => {
    const onSettingsSaved = vi.fn();

    render(
      <SettingsPanel
        isOpen={true}
        onClose={() => {}}
        onSettingsSaved={onSettingsSaved}
        repoName="GitNexus"
      />,
    );

    await screen.findByText('Session Settings');

    const maxTokensInput = screen.getByLabelText('Max Tokens');
    fireEvent.change(maxTokensInput, { target: { value: '4096' } });
    fireEvent.click(screen.getByText('Save Settings'));

    await waitFor(() => expect(onSettingsSaved).toHaveBeenCalled());
    expect(loadLocalRuntimeSettings().codex?.maxTokens).toBe(4096);
    expect(screen.getByText('Settings saved')).toBeInTheDocument();
  });
});
