import { act, render, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { setBackendUrl } from '../../src/services/backend-client';
import { AppStateProvider, useAppState } from '../../src/hooks/useAppState';

const readyStatus = {
  provider: 'codex' as const,
  availability: 'ready' as const,
  available: true,
  authenticated: true,
  executablePath: 'bin/codex',
  version: 'test-version',
  runtimeEnvironment: 'wsl2' as const,
  executionMode: 'bypass' as const,
  supportsSse: true,
  supportsCancel: true,
  supportsMcp: true,
  repo: {
    repoName: 'GitNexus',
    state: 'indexed' as const,
    resolvedRepoName: 'GitNexus',
    resolvedRepoPath: 'repos/GitNexus',
  },
};

let appState: ReturnType<typeof useAppState> | null = null;

function Harness() {
  appState = useAppState();
  return null;
}

describe('useAppState compatibility wrapper', () => {
  beforeEach(() => {
    appState = null;
    setBackendUrl('http://localhost:4747');
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify(readyStatus), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    );
    vi.stubGlobal('requestAnimationFrame', ((callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    }) as typeof requestAnimationFrame);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('preserves the legacy import path while using the local session runtime', async () => {
    render(
      <AppStateProvider>
        <Harness />
      </AppStateProvider>,
    );

    await waitFor(() => expect(appState).not.toBeNull());

    await act(async () => {
      await appState!.initializeAgent('GitNexus');
    });

    expect(appState!.isAgentReady).toBe(true);
    expect(appState!.agentError).toBeNull();
  });
});
