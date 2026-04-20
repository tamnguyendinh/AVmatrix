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
  beforeEach(() => {
    vi.clearAllMocks();
    readFileMock.mockResolvedValue('Final answer');
    unlinkMock.mockResolvedValue(undefined);
  });

  it('prefers WSL2 on Windows when WSL Codex is available', async () => {
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

  it('requires WSL2 on Windows when WSL Codex is unavailable', async () => {
    spawnMock.mockImplementation((command: string, args?: string[]) => {
      const child = new MockChildProcess();
      queueMicrotask(() => {
        if (command === 'wsl.exe') {
          child.stderr.emit('data', Buffer.from('codex not installed in WSL'));
          child.emit('exit', 1);
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
    expect(status.available).toBe(false);
    expect(status.message).toContain('WSL2 Codex is required on Windows');
    expect(status.message).toContain('Windows-native Codex execution is not supported in Phase 1');
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
