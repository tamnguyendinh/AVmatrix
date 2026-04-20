import { beforeEach, describe, expect, it, vi } from 'vitest';

const createServerMock = vi.fn();
const exitMock = vi.spyOn(process, 'exit').mockImplementation((code?: string | number | null) => {
  throw new Error(`process.exit:${code ?? 0}`);
});
const errorMock = vi.spyOn(console, 'error').mockImplementation(() => {});

vi.mock('../../src/server/api.js', () => ({
  createServer: (...args: unknown[]) => createServerMock(...args),
}));

describe('serveCommand local-only host policy', () => {
  beforeEach(() => {
    createServerMock.mockReset();
    errorMock.mockClear();
  });

  it('starts normally on loopback hosts', async () => {
    createServerMock.mockResolvedValue(undefined);
    const { serveCommand } = await import('../../src/cli/serve.js');

    await serveCommand({ port: '4747', host: '127.0.0.1' });

    expect(createServerMock).toHaveBeenCalledWith(4747, '127.0.0.1');
  });

  it('rejects non-loopback hosts before creating the server', async () => {
    const { serveCommand } = await import('../../src/cli/serve.js');

    await expect(serveCommand({ host: '0.0.0.0' })).rejects.toThrow('process.exit:1');
    expect(createServerMock).not.toHaveBeenCalled();
    expect(errorMock).toHaveBeenCalledWith(
      expect.stringContaining('Local-only mode only allows loopback hosts'),
    );
  });
});
