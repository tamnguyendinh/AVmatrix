import { render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { AnalyzeProgress } from '../../src/components/AnalyzeProgress';
import {
  setBackendUrl,
  startAnalyze,
  type AnalyzeRequest,
  type JobStatus,
} from '../../src/services/backend-client';

type AnalyzeRequestHasUrl = 'url' extends keyof AnalyzeRequest ? true : false;
type AnalyzeRequestPathOptional = undefined extends AnalyzeRequest['path'] ? true : false;
type JobStatusHasRepoUrl = 'repoUrl' extends keyof JobStatus ? true : false;
type JobStatusAllowsCloning = 'cloning' extends JobStatus['status'] ? true : false;

const analyzeRequestHasUrl: AnalyzeRequestHasUrl = false;
const analyzeRequestPathOptional: AnalyzeRequestPathOptional = false;
const jobStatusHasRepoUrl: JobStatusHasRepoUrl = false;
const jobStatusAllowsCloning: JobStatusAllowsCloning = false;

describe('analyze contract local-only', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('keeps the active analyze contract path-only', () => {
    expect(analyzeRequestHasUrl).toBe(false);
    expect(analyzeRequestPathOptional).toBe(false);
    expect(jobStatusHasRepoUrl).toBe(false);
    expect(jobStatusAllowsCloning).toBe(false);
  });

  it('posts only the local analyze request body', async () => {
    setBackendUrl('http://localhost:4747');

    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ jobId: 'job-1', status: 'queued' }), {
        status: 202,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await startAnalyze({ path: 'repos/avmatrix', force: true });

    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:4747/api/analyze',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      path: 'repos/avmatrix',
      force: true,
    });
  });

  it('renders local-only progress copy without clone-era labels', () => {
    render(
      <AnalyzeProgress
        progress={{ phase: 'analyzing', percent: 0, message: 'Preparing local analysis...' }}
        onCancel={vi.fn()}
      />,
    );

    expect(screen.getByText('Preparing local analysis')).toBeInTheDocument();
    expect(screen.queryByText('Cloning repository')).not.toBeInTheDocument();
    expect(screen.queryByText('Pulling latest')).not.toBeInTheDocument();
  });
});
