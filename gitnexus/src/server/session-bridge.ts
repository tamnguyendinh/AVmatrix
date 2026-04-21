import type express from 'express';
import type { SessionChatRequest, SessionRepoBinding, SessionStreamEvent } from 'avmatrix-shared';
import { RuntimeController } from '../runtime/runtime-controller.js';
import { SessionRuntimeError } from '../runtime/session-adapter.js';

const parseRepoBinding = (req: express.Request): SessionRepoBinding => {
  const repoName =
    typeof req.query.repoName === 'string'
      ? req.query.repoName
      : req.body && typeof req.body === 'object' && typeof req.body.repoName === 'string'
        ? req.body.repoName
        : undefined;

  const repoPath =
    typeof req.query.repoPath === 'string'
      ? req.query.repoPath
      : req.body && typeof req.body === 'object' && typeof req.body.repoPath === 'string'
        ? req.body.repoPath
        : undefined;

  return { repoName, repoPath };
};

const writeSseEvent = (res: express.Response, event: SessionStreamEvent): void => {
  res.write(`event: ${event.type}\n`);
  res.write(`data: ${JSON.stringify(event)}\n\n`);
};

const handleSessionError = (res: express.Response, error: unknown): void => {
  if (error instanceof SessionRuntimeError) {
    res.status(error.status).json({
      code: error.code,
      error: error.message,
      ...(error.details ? { details: error.details } : {}),
    });
    return;
  }

  const message = error instanceof Error ? error.message : String(error);
  res.status(500).json({
    code: 'SESSION_START_FAILED',
    error: message,
  });
};

export const mountSessionBridge = (app: express.Express, runtime: RuntimeController): void => {
  app.get('/api/session/status', async (req, res) => {
    try {
      const status = await runtime.getStatus(parseRepoBinding(req));
      res.json(status);
    } catch (error) {
      handleSessionError(res, error);
    }
  });

  app.post('/api/session/chat', async (req, res) => {
    const body = req.body as SessionChatRequest | undefined;
    if (!body || typeof body.message !== 'string' || body.message.trim().length === 0) {
      res.status(400).json({
        code: 'BAD_REQUEST',
        error: 'Request body must include a non-empty "message"',
      });
      return;
    }

    try {
      const { job } = await runtime.startChat(body);
      res.set({
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        Connection: 'keep-alive',
        'X-Accel-Buffering': 'no',
      });
      res.flushHeaders();

      const heartbeat = setInterval(() => {
        res.write(':heartbeat\n\n');
      }, 15_000);

      const unsubscribe = job.onEvent((event) => {
        writeSseEvent(res, event);
        if (event.type === 'done' || event.type === 'error' || event.type === 'cancelled') {
          clearInterval(heartbeat);
          unsubscribe();
          res.end();
        }
      });

      req.on('close', () => {
        clearInterval(heartbeat);
        unsubscribe();
        runtime.cancelSession(job.id, 'Client disconnected');
      });
    } catch (error) {
      handleSessionError(res, error);
    }
  });

  app.delete('/api/session/:sessionId', (req, res) => {
    const { sessionId } = req.params;
    const cancelled = runtime.cancelSession(sessionId, 'Cancelled by user');
    if (!cancelled) {
      res.status(404).json({
        code: 'SESSION_NOT_FOUND',
        error: `Session "${sessionId}" was not found or is no longer running`,
      });
      return;
    }

    res.json({ sessionId, status: 'cancelled' });
  });
};
