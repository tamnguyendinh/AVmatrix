import { EventEmitter } from 'events';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { SessionJob } from '../../src/runtime/session-adapter.js';

const { spawnMock, readFileMock, unlinkMock } = vi.hoisted(() => {
  return {
    spawnMock: vi.fn(),
    readFileMock: vi.fn(),
    unlinkMock: vi.fn(),
  };
});

vi.mock('child_process', () => ({
  spawn: spawnMock,
}));

vi.mock('fs/promises', () => ({
  default: {
    readFile: readFileMock,
    unlink: unlinkMock,
  },
}));

class MockChildProcess extends EventEmitter {
  stdout = new EventEmitter();
  stderr = new EventEmitter();
  kill = vi.fn(() => true);
}

describe('CodexSessionAdapter', () => {
  const originalWindowsEnv = process.env.GITNEXUS_WINDOWS_SESSION_ENV;

  beforeEach(() => {
    vi.clearAllMocks();
    process.env.GITNEXUS_WINDOWS_SESSION_ENV = 'native';
    readFileMock.mockResolvedValue('Final answer');
    unlinkMock.mockResolvedValue(undefined);
  });

  afterEach(() => {
    if (originalWindowsEnv === undefined) {
      delete process.env.GITNEXUS_WINDOWS_SESSION_ENV;
    } else {
      process.env.GITNEXUS_WINDOWS_SESSION_ENV = originalWindowsEnv;
    }
  });

  it('prefers WSL2 on Windows when WSL Codex is available', async () => {
    process.env.GITNEXUS_WINDOWS_SESSION_ENV = 'auto';

    spawnMock.mockImplementation((command: string, args?: string[]) => {
      const child = new MockChildProcess();
      queueMicrotask(() => {
        if (command === 'wsl.exe') {
          const commandLine = Array.isArray(args) ? args.join(' ') : '';
          if (commandLine.includes('--version')) {
            child.stdout.emit('data', Buffer.from('codex-cli 0.119.0\n'));
          } else {
            child.stdout.emit('data', Buffer.from('Logged in using ChatGPT\n'));
          }
          child.emit('exit', 0);
          return;
        }
        child.emit('exit', 1);
      });
      return child;
    });

    const { CodexSessionAdapter } = await import('../../src/runtime/session-adapters/codex.js');
    const adapter = new CodexSessionAdapter();
    const status = await adapter.getStatus();

    expect(status.runtimeEnvironment).toBe('wsl2');
    expect(status.available).toBe(true);
  });

  it('falls back to native on Windows when WSL Codex is unavailable but native Codex is ready', async () => {
    process.env.GITNEXUS_WINDOWS_SESSION_ENV = 'auto';

    spawnMock.mockImplementation((command: string, args?: string[]) => {
      const child = new MockChildProcess();
      queueMicrotask(() => {
        if (command === 'wsl.exe') {
          child.stderr.emit('data', Buffer.from('codex not installed in WSL'));
          child.emit('exit', 1);
          return;
        }

        const commandLine = Array.isArray(args) ? args.join(' ') : '';
        if (commandLine.includes('--version')) {
          child.stdout.emit('data', Buffer.from('codex-cli 0.119.0\n'));
        } else {
          child.stdout.emit('data', Buffer.from('Logged in using ChatGPT\n'));
        }
        child.emit('exit', 0);
      });
      return child;
    });

    const { CodexSessionAdapter } = await import('../../src/runtime/session-adapters/codex.js');
    const adapter = new CodexSessionAdapter();
    const status = await adapter.getStatus();

    expect(status.runtimeEnvironment).toBe('native');
    expect(status.available).toBe(true);
    expect(status.message).toContain('WSL2 Codex unavailable; using native fallback');
  });

  it('maps agent messages to reasoning and command aggregated_output to tool results', async () => {
    const versionChild = new MockChildProcess();
    const loginChild = new MockChildProcess();
    const execChild = new MockChildProcess();

    spawnMock
      .mockImplementationOnce(() => {
        queueMicrotask(() => {
          versionChild.stdout.emit('data', Buffer.from('codex-cli 0.119.0\n'));
          versionChild.emit('exit', 0);
        });
        return versionChild;
      })
      .mockImplementationOnce(() => {
        queueMicrotask(() => {
          loginChild.stdout.emit('data', Buffer.from('Logged in using ChatGPT\n'));
          loginChild.emit('exit', 0);
        });
        return loginChild;
      })
      .mockImplementationOnce(() => {
        queueMicrotask(() => {
          execChild.stdout.emit(
            'data',
            Buffer.from(
              [
                JSON.stringify({
                  type: 'item.started',
                  item: { id: 'cmd-1', type: 'command_execution', command: ['dir'] },
                }),
                JSON.stringify({
                  type: 'item.completed',
                  item: {
                    id: 'cmd-1',
                    type: 'command_execution',
                    command: ['dir'],
                    aggregated_output: 'file-a\nfile-b',
                  },
                }),
                JSON.stringify({
                  type: 'item.completed',
                  item: { id: 'msg-1', type: 'agent_message', text: 'Thinking step' },
                }),
                JSON.stringify({
                  type: 'turn.completed',
                  usage: { output_tokens: 5 },
                }),
              ].join('\n') + '\n',
            ),
          );
          execChild.emit('exit', 0);
        });
        return execChild;
      });

    const { CodexSessionAdapter } = await import('../../src/runtime/session-adapters/codex.js');
    const adapter = new CodexSessionAdapter();
    const job = new SessionJob('codex', 'demo', 'F:/repo', new AbortController());
    const events: any[] = [];
    const completion = new Promise<void>((resolve) => {
      job.onEvent((event) => {
        if (event.type === 'done' || event.type === 'error' || event.type === 'cancelled') {
          resolve();
        }
      }, false);
    });
    job.onEvent((event) => {
      events.push(event);
    });

    await adapter.runChat(
      job,
      { repoPath: 'F:/repo', message: 'hello' },
      { repo: { repoName: 'demo', repoPath: 'F:/repo', indexed: true } },
      job.signal,
    );
    await completion;

    expect(events.some((event) => event.type === 'reasoning' && event.reasoning === 'Thinking step')).toBe(true);
    expect(
      events.some(
        (event) =>
          event.type === 'tool_result' &&
          event.toolCall?.result === 'file-a\nfile-b',
      ),
    ).toBe(true);
    expect(events.some((event) => event.type === 'content' && event.content === 'Final answer')).toBe(true);
    expect(events.some((event) => event.type === 'done')).toBe(true);
  });
});
