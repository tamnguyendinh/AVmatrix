import express from 'express';
import type { Server } from 'http';
import type { AddressInfo } from 'net';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { SessionStatusResponse } from 'gitnexus-shared';
import { SessionJob, SessionRuntimeError } from '../../src/runtime/session-adapter.js';
import { mountSessionBridge } from '../../src/server/session-bridge.js';

interface StartChatResult {
  job: SessionJob;
  repo: {
    repoName: string;
    repoPath: string;
    indexed: boolean;
  };
}

class FakeRuntimeController {
  statusResponse: SessionStatusResponse = {
    provider: 'codex',
    availability: 'ready',
    available: true,
    authenticated: true,
    executablePath: 'codex.cmd',
    version: 'codex-cli test',
    recommendedEnvironment: 'wsl2',
    executionMode: 'bypass',
    supportsSse: true,
    supportsCancel: true,
    supportsMcp: true,
  };

  startChatImpl: (message: string) => Promise<StartChatResult> = async (message) => {
    const job = new SessionJob('codex', 'demo', 'C:/demo', new AbortController());
    job.emit({
      sessionId: job.id,
      provider: 'codex',
      repoName: 'demo',
      repoPath: 'C:/demo',
      timestamp: Date.now(),
      type: 'session_started',
      executionMode: 'bypass',
    });
    setTimeout(() => {
      job.emit({
        sessionId: job.id,
        provider: 'codex',
        repoName: 'demo',
        repoPath: 'C:/demo',
        timestamp: Date.now(),
        type: 'content',
        content: `echo:${message}`,
      });
      job.emit({
        sessionId: job.id,
        provider: 'codex',
        repoName: 'demo',
        repoPath: 'C:/demo',
        timestamp: Date.now(),
        type: 'done',
      });
    }, 10);

    return {
      job,
      repo: {
        repoName: 'demo',
        repoPath: 'C:/demo',
        indexed: true,
      },
    };
  };

  cancelResult = true;
  readonly getStatus = vi.fn(async () => this.statusResponse);
  readonly startChat = vi.fn(async (request: { message: string }) => this.startChatImpl(request.message));
  readonly cancelSession = vi.fn((_sessionId: string, _reason?: string) => this.cancelResult);
}

describe('session bridge', () => {
  const servers: Server[] = [];

  const startApp = async (runtime: FakeRuntimeController) => {
    const app = express();
    app.use(express.json());
    mountSessionBridge(app, runtime as any);

    const server = await new Promise<Server>((resolve) => {
      const instance = app.listen(0, () => resolve(instance));
    });
    servers.push(server);
    const address = server.address() as AddressInfo;
    return `http://127.0.0.1:${address.port}`;
  };

  afterEach(async () => {
    await Promise.allSettled(
      servers.splice(0).map(
        (server) =>
          new Promise<void>((resolve, reject) => {
            server.close((error) => (error ? reject(error) : resolve()));
          }),
      ),
    );
  });

  it('returns session status and repo binding state', async () => {
    const runtime = new FakeRuntimeController();
    runtime.statusResponse = {
      ...runtime.statusResponse,
      repo: {
        repoName: 'demo',
        repoPath: 'C:/demo',
        state: 'indexed',
        resolvedRepoName: 'demo',
        resolvedRepoPath: 'C:/demo',
      },
    };
    const baseUrl = await startApp(runtime);

    const response = await fetch(`${baseUrl}/api/session/status?repoName=demo`);
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(runtime.getStatus).toHaveBeenCalledWith({ repoName: 'demo', repoPath: undefined });
    expect(body).toMatchObject({
      provider: 'codex',
      available: true,
      repo: {
        state: 'indexed',
        resolvedRepoName: 'demo',
      },
    });
  });

  it('rejects chat requests without a message', async () => {
    const runtime = new FakeRuntimeController();
    const baseUrl = await startApp(runtime);

    const response = await fetch(`${baseUrl}/api/session/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ repoName: 'demo' }),
    });
    const body = await response.json();

    expect(response.status).toBe(400);
    expect(body.code).toBe('BAD_REQUEST');
    expect(runtime.startChat).not.toHaveBeenCalled();
  });

  it('streams session_started, content, and done events over SSE', async () => {
    const runtime = new FakeRuntimeController();
    const baseUrl = await startApp(runtime);

    const response = await fetch(`${baseUrl}/api/session/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ repoName: 'demo', message: 'hello' }),
    });
    const body = await response.text();

    expect(response.status).toBe(200);
    expect(response.headers.get('content-type')).toContain('text/event-stream');
    expect(body).toContain('event: session_started');
    expect(body).toContain('event: content');
    expect(body).toContain('event: done');
    expect(body).toContain('"content":"echo:hello"');
  });

  it('surfaces INDEX_REQUIRED errors without starting a stream', async () => {
    const runtime = new FakeRuntimeController();
    runtime.startChatImpl = async () => {
      throw new SessionRuntimeError(
        'INDEX_REQUIRED',
        'Repository is not indexed yet. Run analyze first.',
        409,
        { repoPath: 'C:/demo' },
      );
    };
    const baseUrl = await startApp(runtime);

    const response = await fetch(`${baseUrl}/api/session/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ repoPath: 'C:/demo', message: 'hello' }),
    });
    const body = await response.json();

    expect(response.status).toBe(409);
    expect(body).toMatchObject({
      code: 'INDEX_REQUIRED',
      error: 'Repository is not indexed yet. Run analyze first.',
    });
  });

  it('cancels known sessions and returns 404 for unknown sessions', async () => {
    const runtime = new FakeRuntimeController();
    const baseUrl = await startApp(runtime);

    const okResponse = await fetch(`${baseUrl}/api/session/session-123`, {
      method: 'DELETE',
    });
    const okBody = await okResponse.json();

    expect(okResponse.status).toBe(200);
    expect(okBody).toEqual({ sessionId: 'session-123', status: 'cancelled' });
    expect(runtime.cancelSession).toHaveBeenCalledWith('session-123', 'Cancelled by user');

    runtime.cancelResult = false;

    const notFoundResponse = await fetch(`${baseUrl}/api/session/missing`, {
      method: 'DELETE',
    });
    const notFoundBody = await notFoundResponse.json();

    expect(notFoundResponse.status).toBe(404);
    expect(notFoundBody.code).toBe('SESSION_NOT_FOUND');
  });
});
