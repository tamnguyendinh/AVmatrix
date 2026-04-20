import path from 'path';
import fs from 'fs/promises';
import os from 'os';
import { randomUUID } from 'crypto';
import { spawn } from 'child_process';
import type {
  SessionChatRequest,
  SessionExecutionMode,
  SessionRuntimeEnvironment,
  SessionStatus,
  SessionToolCall,
  SessionStreamEvent,
} from 'gitnexus-shared';
import type { SessionAdapter, SessionChatContext, SessionJob } from '../session-adapter.js';
import { SessionRuntimeError } from '../session-adapter.js';

const COMMAND_TIMEOUT_MS = 10_000;
const STDERR_LIMIT = 8_192;

interface ProcessResult {
  code: number;
  stdout: string;
  stderr: string;
}

interface CodexLaunchTarget {
  runtimeEnvironment: SessionRuntimeEnvironment;
  executablePath: string;
  args: string[];
}

interface CodexStatusProbe {
  target: CodexLaunchTarget;
  available: boolean;
  authenticated: boolean;
  version?: string;
  message?: string;
}

const getNativeCodexExecutable = (): string =>
  process.env.GITNEXUS_CODEX_EXECUTABLE ||
  (process.platform === 'win32' ? 'codex.cmd' : 'codex');

const getConfiguredExecutionMode = (): SessionExecutionMode => {
  const configured = process.env.GITNEXUS_SESSION_EXECUTION_MODE;
  if (configured === 'sandbox' || configured === 'sandboxed') return 'sandboxed';
  if (configured === 'bypass') return 'bypass';
  return process.platform === 'win32' ? 'bypass' : 'sandboxed';
};

const shellQuotePosix = (value: string): string => `'${value.replace(/'/g, `'\\''`)}'`;

const toWslPath = (value: string): string => {
  if (/^[a-z]:\\/i.test(value)) {
    const drive = value[0].toLowerCase();
    const rest = value.slice(2).replace(/\\/g, '/');
    return `/mnt/${drive}${rest}`;
  }
  return value.replace(/\\/g, '/');
};

const spawnCommand = (
  target: CodexLaunchTarget,
  options: Parameters<typeof spawn>[2] = {},
) =>
  spawn(target.executablePath, target.args, {
    ...options,
    shell: false,
    windowsHide: true,
  });

const runCommand = async (
  target: CodexLaunchTarget,
  timeoutMs = COMMAND_TIMEOUT_MS,
): Promise<ProcessResult> =>
  new Promise((resolve, reject) => {
    const child = spawnCommand(target, {
      stdio: ['ignore', 'pipe', 'pipe'],
    });

    let stdout = '';
    let stderr = '';
    const timeout = setTimeout(() => {
      child.kill('SIGTERM');
      reject(new Error(`Command timed out after ${timeoutMs}ms`));
    }, timeoutMs);

    child.stdout.on('data', (chunk: Buffer) => {
      stdout += chunk.toString();
    });
    child.stderr.on('data', (chunk: Buffer) => {
      stderr += chunk.toString();
    });

    child.once('error', (error) => {
      clearTimeout(timeout);
      reject(error);
    });

    child.once('exit', (code) => {
      clearTimeout(timeout);
      resolve({ code: code ?? 1, stdout, stderr });
    });
  });

const buildNativeTarget = (...args: string[]): CodexLaunchTarget => ({
  runtimeEnvironment: 'native',
  executablePath: getNativeCodexExecutable(),
  args,
});

const buildWslTarget = (...args: string[]): CodexLaunchTarget => ({
  runtimeEnvironment: 'wsl2',
  executablePath: 'wsl.exe',
  args: ['-e', 'bash', '-lc', args.map(shellQuotePosix).join(' ')],
});

const createBaseEvent = (
  job: SessionJob,
): Pick<SessionStreamEvent, 'sessionId' | 'provider' | 'repoName' | 'repoPath' | 'timestamp'> => ({
  sessionId: job.id,
  provider: job.provider,
  repoName: job.repoName,
  repoPath: job.repoPath,
  timestamp: Date.now(),
});

const coerceToolCall = (
  item: Record<string, unknown>,
  status: SessionToolCall['status'],
): SessionToolCall | null => {
  const itemType = typeof item.type === 'string' ? item.type : '';
  if (itemType !== 'command_execution' && itemType !== 'mcp_tool_call') {
    return null;
  }

  const id = typeof item.id === 'string' ? item.id : randomUUID();
  const command =
    typeof item.command === 'string'
      ? item.command
      : Array.isArray(item.command)
        ? item.command.map((part) => String(part)).join(' ')
        : undefined;
  const name =
    itemType === 'mcp_tool_call'
      ? String(item.name || item.tool_name || 'mcp_tool_call')
      : command || String(item.name || 'command_execution');

  const resultParts = [
    typeof item.aggregated_output === 'string' ? item.aggregated_output : '',
    typeof item.stdout === 'string' ? item.stdout : '',
    typeof item.stderr === 'string' ? item.stderr : '',
    typeof item.output === 'string' ? item.output : '',
    typeof item.text === 'string' ? item.text : '',
  ].filter(Boolean);

  return {
    id,
    name,
    args: command ? { command } : undefined,
    result: resultParts.length > 0 ? resultParts.join('\n').trim() : undefined,
    status,
  };
};

const emitCodexEvent = (
  job: SessionJob,
  payload: Record<string, unknown>,
  lastReasoning: { value: string },
  completionUsage: { value?: Record<string, number> },
) => {
  const eventType = typeof payload.type === 'string' ? payload.type : '';

  if (eventType === 'item.started' || eventType === 'item.completed') {
    const item = payload.item;
    if (item && typeof item === 'object') {
      const toolCall = coerceToolCall(
        item as Record<string, unknown>,
        eventType === 'item.started' ? 'running' : 'completed',
      );
      if (toolCall) {
        job.emit({
          ...createBaseEvent(job),
          type: eventType === 'item.started' ? 'tool_call' : 'tool_result',
          toolCall,
        });
        return;
      }

      const itemType =
        typeof (item as Record<string, unknown>).type === 'string'
          ? String((item as Record<string, unknown>).type)
          : '';
      if (itemType === 'agent_message') {
        const text =
          typeof (item as Record<string, unknown>).text === 'string'
            ? String((item as Record<string, unknown>).text)
            : '';
        if (text) {
          lastReasoning.value = text;
          job.emit({
            ...createBaseEvent(job),
            type: 'reasoning',
            reasoning: text,
          });
        }
      }
    }
    return;
  }

  if (eventType === 'agent_message') {
    const text =
      typeof payload.text === 'string'
        ? payload.text
        : payload.message &&
            typeof payload.message === 'object' &&
            typeof (payload.message as Record<string, unknown>).text === 'string'
          ? String((payload.message as Record<string, unknown>).text)
          : '';
    if (text) {
      lastReasoning.value = text;
      job.emit({
        ...createBaseEvent(job),
        type: 'reasoning',
        reasoning: text,
      });
    }
    return;
  }

  if (eventType === 'turn.completed') {
    completionUsage.value =
      payload.usage && typeof payload.usage === 'object'
        ? Object.fromEntries(
            Object.entries(payload.usage as Record<string, unknown>).filter(([, value]) =>
              typeof value === 'number',
            ) as [string, number][],
          )
        : undefined;
  }
};

const probeTargetStatus = async (target: CodexLaunchTarget): Promise<CodexStatusProbe> => {
  try {
    const versionResult = await runCommand(
      target.runtimeEnvironment === 'wsl2'
        ? buildWslTarget('codex', '--version')
        : buildNativeTarget('--version'),
    );
    if (versionResult.code !== 0) {
      return {
        target,
        available: false,
        authenticated: false,
        message: versionResult.stderr || versionResult.stdout || 'codex --version failed',
      };
    }

    const loginResult = await runCommand(
      target.runtimeEnvironment === 'wsl2'
        ? buildWslTarget('codex', 'login', 'status')
        : buildNativeTarget('login', 'status'),
    );
    const loginOutput = `${loginResult.stdout}\n${loginResult.stderr}`.trim();
    const authenticated = /logged in/i.test(loginOutput) && !/not logged in/i.test(loginOutput);

    return {
      target,
      available: true,
      authenticated,
      version: versionResult.stdout.trim() || versionResult.stderr.trim() || undefined,
      message: authenticated ? undefined : loginOutput || 'Codex CLI is installed but not signed in',
    };
  } catch (error) {
    return {
      target,
      available: false,
      authenticated: false,
      message: error instanceof Error ? error.message : String(error),
    };
  }
};

const resolveWindowsTarget = async (): Promise<CodexStatusProbe> => {
  const wslTarget = buildWslTarget();

  const wslStatus = await probeTargetStatus(wslTarget);
  if (wslStatus.available) {
    return wslStatus;
  }

  return {
    target: wslTarget,
    available: false,
    authenticated: false,
    message: [
      'WSL2 Codex is required on Windows for the local session runtime.',
      'Windows-native Codex execution is not supported in Phase 1.',
      'Install Codex CLI inside WSL2.',
      wslStatus.message,
    ]
      .filter(Boolean)
      .join(' '),
  };
};

const resolveLaunchTarget = async (): Promise<CodexStatusProbe> => {
  if (process.platform === 'win32') {
    return resolveWindowsTarget();
  }
  const nativeStatus = await probeTargetStatus(buildNativeTarget());
  return nativeStatus;
};

export class CodexSessionAdapter implements SessionAdapter {
  readonly provider = 'codex' as const;
  readonly executionMode = getConfiguredExecutionMode();
  runtimeEnvironment: SessionRuntimeEnvironment = process.platform === 'win32' ? 'wsl2' : 'native';

  async getStatus(): Promise<SessionStatus> {
    const probe = await resolveLaunchTarget();
    this.runtimeEnvironment = probe.target.runtimeEnvironment;

    return {
      provider: this.provider,
      availability: probe.available ? (probe.authenticated ? 'ready' : 'not_signed_in') : 'not_installed',
      available: probe.available,
      authenticated: probe.authenticated,
      executablePath:
        probe.target.runtimeEnvironment === 'wsl2'
          ? 'wsl.exe -> codex'
          : probe.target.executablePath,
      version: probe.version,
      message: probe.message,
      recommendedEnvironment: process.platform === 'win32' ? 'wsl2' : 'native',
      runtimeEnvironment: probe.target.runtimeEnvironment,
      executionMode: this.executionMode,
      supportsSse: true,
      supportsCancel: true,
      supportsMcp: true,
    };
  }

  async runChat(
    job: SessionJob,
    request: SessionChatRequest,
    context: SessionChatContext,
    signal: AbortSignal,
  ): Promise<void> {
    const status = await this.getStatus();
    if (!status.available) {
      throw new SessionRuntimeError(
        'SESSION_RUNTIME_UNAVAILABLE',
        status.message || 'Codex CLI is not available',
        503,
      );
    }
    if (!status.authenticated) {
      throw new SessionRuntimeError(
        'SESSION_NOT_SIGNED_IN',
        status.message || 'Codex CLI is not signed in',
        401,
      );
    }

    const outputFile = path.join(os.tmpdir(), `gitnexus-codex-${job.id}.txt`);
    const runtimeRepoPath =
      status.runtimeEnvironment === 'wsl2' ? toWslPath(context.repo.repoPath) : context.repo.repoPath;
    const runtimeOutputPath =
      status.runtimeEnvironment === 'wsl2' ? toWslPath(outputFile) : outputFile;

    const baseArgs = [
      'exec',
      '--json',
      '--skip-git-repo-check',
      '--output-last-message',
      runtimeOutputPath,
      '--cd',
      runtimeRepoPath,
    ];

    if (this.executionMode === 'bypass') {
      baseArgs.push('--dangerously-bypass-approvals-and-sandbox');
    } else {
      baseArgs.push('--full-auto');
    }

    baseArgs.push(request.message);

    const target =
      status.runtimeEnvironment === 'wsl2'
        ? buildWslTarget('codex', ...baseArgs)
        : buildNativeTarget(...baseArgs);

    const child = spawnCommand(target, {
      cwd: context.repo.repoPath,
      stdio: ['ignore', 'pipe', 'pipe'],
    });

    let stdoutBuffer = '';
    let stderrTail = '';
    let terminalEmitted = false;
    const lastReasoning = { value: '' };
    const completionUsage: { value?: Record<string, number> } = {};

    const cleanup = async () => {
      signal.removeEventListener('abort', onAbort);
      try {
        await fs.unlink(outputFile);
      } catch {}
    };

    const onAbort = () => {
      child.kill('SIGTERM');
    };

    signal.addEventListener('abort', onAbort, { once: true });

    child.stdout.on('data', (chunk: Buffer) => {
      stdoutBuffer += chunk.toString();
      const lines = stdoutBuffer.split(/\r?\n/);
      stdoutBuffer = lines.pop() ?? '';

      for (const rawLine of lines) {
        const line = rawLine.trim();
        if (!line) continue;
        try {
          const payload = JSON.parse(line) as Record<string, unknown>;
          emitCodexEvent(job, payload, lastReasoning, completionUsage);
        } catch {
          // Ignore non-JSON lines from Codex CLI.
        }
      }
    });

    child.stderr.on('data', (chunk: Buffer) => {
      stderrTail += chunk.toString();
      if (stderrTail.length > STDERR_LIMIT) {
        stderrTail = stderrTail.slice(-STDERR_LIMIT);
      }
    });

    child.once('error', async (error) => {
      if (terminalEmitted) return;
      job.emit({
        ...createBaseEvent(job),
        type: 'error',
        code: 'SESSION_START_FAILED',
        error: error.message,
      });
      terminalEmitted = true;
      await cleanup();
    });

    child.once('exit', async (code) => {
      const aborted = signal.aborted;
      if (!terminalEmitted) {
        if (aborted) {
          job.emit({
            ...createBaseEvent(job),
            type: 'cancelled',
            reason: typeof signal.reason === 'string' ? signal.reason : 'Session cancelled',
          });
        } else if (code === 0) {
          let finalContent = '';
          try {
            finalContent = (await fs.readFile(outputFile, 'utf-8')).trim();
          } catch {}
          if (!finalContent && lastReasoning.value) {
            finalContent = lastReasoning.value;
          }
          if (finalContent) {
            job.emit({
              ...createBaseEvent(job),
              type: 'content',
              content: finalContent,
            });
          }
          job.emit({
            ...createBaseEvent(job),
            type: 'done',
            usage: completionUsage.value,
          });
        } else {
          job.emit({
            ...createBaseEvent(job),
            type: 'error',
            code: 'SESSION_START_FAILED',
            error: stderrTail.trim() || `Codex exited with code ${code ?? 'unknown'}`,
          });
        }
      }

      await cleanup();
    });
  }
}
