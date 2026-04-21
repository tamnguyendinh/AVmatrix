import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const {
  databaseCtorMock,
  connectionCtorMock,
  queryMock,
  closeConnectionMock,
  closeDatabaseMock,
  lstatMock,
  mkdirMock,
  unlinkMock,
} = vi.hoisted(() => ({
  databaseCtorMock: vi.fn(),
  connectionCtorMock: vi.fn(),
  queryMock: vi.fn(),
  closeConnectionMock: vi.fn(),
  closeDatabaseMock: vi.fn(),
  lstatMock: vi.fn(),
  mkdirMock: vi.fn(),
  unlinkMock: vi.fn(),
}));

vi.mock('@ladybugdb/core', () => ({
  default: {
    Database: function Database(dbPath: string) {
      databaseCtorMock(dbPath);
      return {
        close: closeDatabaseMock,
      };
    },
    Connection: function Connection(_db: unknown) {
      connectionCtorMock();
      return {
        query: queryMock,
        close: closeConnectionMock,
      };
    },
  },
}));

vi.mock('fs/promises', () => ({
  default: {
    lstat: lstatMock,
    mkdir: mkdirMock,
    unlink: unlinkMock,
  },
}));

describe('lbug WAL recovery', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    lstatMock.mockRejectedValue(Object.assign(new Error('ENOENT'), { code: 'ENOENT' }));
    mkdirMock.mockResolvedValue(undefined);
    unlinkMock.mockResolvedValue(undefined);
    closeConnectionMock.mockResolvedValue(undefined);
    closeDatabaseMock.mockResolvedValue(undefined);
  });

  afterEach(async () => {
    try {
      const adapter = await import('../../src/core/lbug/lbug-adapter.js');
      await adapter.closeLbug();
    } catch {}
  });

  it('removes .wal and .lock sidecars then retries when schema creation hits WAL corruption', async () => {
    queryMock
      .mockRejectedValueOnce(
        new Error('Runtime exception: Corrupted wal file. Read out invalid WAL record type.'),
      )
      .mockResolvedValue([]);

    const adapter = await import('../../src/core/lbug/lbug-adapter.js');

    await expect(adapter.initLbug('repos/demo/.gitnexus/lbug')).resolves.toBeDefined();

    expect(databaseCtorMock).toHaveBeenCalledTimes(2);
    expect(connectionCtorMock).toHaveBeenCalledTimes(2);
    expect(unlinkMock).toHaveBeenCalledWith('repos/demo/.gitnexus/lbug.wal');
    expect(unlinkMock).toHaveBeenCalledWith('repos/demo/.gitnexus/lbug.lock');
  });

  it('retries initialization when VECTOR extension load hits WAL corruption', async () => {
    let shouldCorruptVectorLoad = true;
    queryMock.mockImplementation(async (query: string) => {
      if (query === 'INSTALL VECTOR' && shouldCorruptVectorLoad) {
        shouldCorruptVectorLoad = false;
        throw new Error('Runtime exception: Corrupted wal file. Read out invalid WAL record type.');
      }
      return [];
    });

    const adapter = await import('../../src/core/lbug/lbug-adapter.js');

    await expect(adapter.initLbug('repos/vector/.gitnexus/lbug')).resolves.toBeDefined();

    expect(databaseCtorMock).toHaveBeenCalledTimes(2);
    expect(unlinkMock).toHaveBeenCalledWith('repos/vector/.gitnexus/lbug.wal');
    expect(unlinkMock).toHaveBeenCalledWith('repos/vector/.gitnexus/lbug.lock');
  });
});
