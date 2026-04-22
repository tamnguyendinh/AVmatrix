import { beforeEach, describe, expect, it, vi } from 'vitest';

const initMock = vi.fn();
const listReposMock = vi.fn();
const startMCPServerMock = vi.fn();
const stderrWriteMock = vi.spyOn(process.stderr, 'write').mockImplementation(() => true);

vi.mock('../../src/mcp/server.js', () => ({
  startMCPServer: (...args: unknown[]) => startMCPServerMock(...args),
}));

vi.mock('../../src/mcp/local/local-backend.js', () => ({
  LocalBackend: class LocalBackend {
    init = initMock;
    listRepos = listReposMock;
  },
}));

describe('mcpCommand runtime alignment', () => {
  beforeEach(() => {
    initMock.mockReset().mockResolvedValue(true);
    listReposMock.mockReset().mockResolvedValue([{ name: 'AVmatrix' }]);
    startMCPServerMock.mockReset().mockResolvedValue(undefined);
    stderrWriteMock.mockClear();
  });

  it('connects the MCP transport before any repo discovery work', async () => {
    const { mcpCommand } = await import('../../src/cli/mcp.js');

    await mcpCommand();

    expect(startMCPServerMock).toHaveBeenCalledTimes(1);
    expect(initMock).not.toHaveBeenCalled();
    expect(listReposMock).not.toHaveBeenCalled();
    expect(stderrWriteMock).toHaveBeenCalledWith(
      expect.stringContaining('stage=backend_created'),
    );
    expect(stderrWriteMock).toHaveBeenCalledWith(
      expect.stringContaining('stage=transport_connected'),
    );
  });

  it('does not block startup on repo discovery availability', async () => {
    const { mcpCommand } = await import('../../src/cli/mcp.js');

    await mcpCommand();

    expect(startMCPServerMock).toHaveBeenCalledTimes(1);
    expect(initMock).not.toHaveBeenCalled();
    expect(listReposMock).not.toHaveBeenCalled();
  });
});
