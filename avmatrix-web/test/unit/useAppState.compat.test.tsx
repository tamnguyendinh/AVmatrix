import { act, render, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AppStateProvider, useAppState } from '../../src/hooks/useAppState';

let appState: ReturnType<typeof useAppState> | null = null;

function Harness() {
  appState = useAppState();
  return null;
}

describe('useAppState compatibility wrapper', () => {
  beforeEach(() => {
    appState = null;
    vi.stubGlobal('requestAnimationFrame', ((callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    }) as typeof requestAnimationFrame);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('preserves the legacy import path while exposing the chat runtime bridge', async () => {
    render(
      <AppStateProvider>
        <Harness />
      </AppStateProvider>,
    );

    await waitFor(() => expect(appState).not.toBeNull());

    expect(appState!.chatRuntimeBridge.getRepoName()).toBeUndefined();

    await act(async () => {
      appState!.setProjectName('avmatrix');
    });

    expect(appState!.chatRuntimeBridge.getRepoName()).toBe('avmatrix');
    expect(appState!.chatRuntimeBridge.getEmbeddingStatus()).toBe('idle');
  });
});
