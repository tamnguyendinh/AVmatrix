import { beforeEach, describe, expect, it, vi } from 'vitest';

const initMock = vi.fn();
const listReposMock = vi.fn();
const startMCPServerMock = vi.fn();
const errorMock = vi.spyOn(console, 'error').mockImplementation(() => {});

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
    listReposMock.mockReset().mockResolvedValue([{ name: 'GitNexus' }]);
    startMCPServerMock.mockReset().mockResolvedValue(undefined);
    errorMock.mockClear();
  });

  it('boots the MCP surface on the shared local runtime core', async () => {
    const { mcpCommand } = await import('../../src/cli/mcp.js');

    await mcpCommand();

    expect(initMock).toHaveBeenCalledTimes(1);
    expect(listReposMock).toHaveBeenCalledTimes(1);
    expect(startMCPServerMock).toHaveBeenCalledTimes(1);
    expect(errorMock).toHaveBeenCalledWith(
      expect.stringContaining('shared local runtime core'),
    );
  });

  it('starts even when no indexed repos are present and prints the analyze hint', async () => {
    listReposMock.mockResolvedValueOnce([]);
    const { mcpCommand } = await import('../../src/cli/mcp.js');

    await mcpCommand();

    expect(startMCPServerMock).toHaveBeenCalledTimes(1);
    expect(errorMock).toHaveBeenCalledWith(
      expect.stringContaining('Run `gitnexus analyze` in a local repo'),
    );
  });
});
