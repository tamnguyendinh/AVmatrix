import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useBackend } from '../../src/hooks/useBackend';
import { DEFAULT_BACKEND_URL } from '../../src/config/ui-constants';

const probeBackendMock = vi.fn();
const setBackendUrlMock = vi.fn();

vi.mock('../../src/services/backend-client', () => ({
  normalizeServerUrl: (input: string) => {
    const trimmed = input.trim().replace(/\/+$/, '');
    if (trimmed === 'localhost:4747' || trimmed === DEFAULT_BACKEND_URL) {
      return DEFAULT_BACKEND_URL;
    }
    throw new Error(
      'GitNexus local-only mode only supports backend URLs on localhost, 127.0.0.1, or [::1].',
    );
  },
  probeBackend: (...args: unknown[]) => probeBackendMock(...args),
  setBackendUrl: (...args: unknown[]) => setBackendUrlMock(...args),
}));

function TestProbe() {
  const { backendUrl, isConnected } = useBackend();
  return (
    <div>
      <span data-testid="backend-url">{backendUrl}</span>
      <span data-testid="backend-connected">{String(isConnected)}</span>
    </div>
  );
}

describe('useBackend local-only migration', () => {
  beforeEach(() => {
    probeBackendMock.mockReset();
    setBackendUrlMock.mockReset();
    localStorage.clear();
    probeBackendMock.mockResolvedValue(true);
  });

  it('falls back to the default local backend when legacy storage contains a remote host', async () => {
    localStorage.setItem('gitnexus-backend-url', 'https://gitnexus.example.com');

    render(<TestProbe />);

    expect(screen.getByTestId('backend-url').textContent).toBe(DEFAULT_BACKEND_URL);
    await waitFor(() => expect(setBackendUrlMock).toHaveBeenCalledWith(DEFAULT_BACKEND_URL));
    expect(localStorage.getItem('gitnexus-backend-url')).toBeNull();
  });

  it('keeps a normalized local loopback backend from storage', async () => {
    localStorage.setItem('gitnexus-backend-url', 'localhost:4747');

    render(<TestProbe />);

    expect(screen.getByTestId('backend-url').textContent).toBe(DEFAULT_BACKEND_URL);
    await waitFor(() => expect(setBackendUrlMock).toHaveBeenCalledWith(DEFAULT_BACKEND_URL));
    expect(localStorage.getItem('gitnexus-backend-url')).toBe(DEFAULT_BACKEND_URL);
  });
});
