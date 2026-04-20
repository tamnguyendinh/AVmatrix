import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { RepoAnalyzer } from '../../src/components/RepoAnalyzer';

const startAnalyzeMock = vi.fn();
const cancelAnalyzeMock = vi.fn();
const streamAnalyzeProgressMock = vi.fn();

vi.mock('../../src/services/backend-client', () => ({
  startAnalyze: (...args: unknown[]) => startAnalyzeMock(...args),
  cancelAnalyze: (...args: unknown[]) => cancelAnalyzeMock(...args),
  streamAnalyzeProgress: (...args: unknown[]) => streamAnalyzeProgressMock(...args),
}));

vi.mock('../../src/components/AnalyzeProgress', () => ({
  AnalyzeProgress: () => <div>Analyzing</div>,
}));

describe('RepoAnalyzer local-only', () => {
  beforeEach(() => {
    startAnalyzeMock.mockReset();
    cancelAnalyzeMock.mockReset();
    streamAnalyzeProgressMock.mockReset();
    startAnalyzeMock.mockResolvedValue({ jobId: 'job-1', status: 'queued' });
    streamAnalyzeProgressMock.mockReturnValue(new AbortController());
  });

  it('shows only local-folder input', () => {
    render(<RepoAnalyzer variant="sheet" onComplete={vi.fn()} />);

    expect(screen.getByText('Local Folder Path')).toBeInTheDocument();
    expect(screen.queryByText('GitHub URL')).not.toBeInTheDocument();
    expect(screen.queryByText('GitHub Repository URL')).not.toBeInTheDocument();
  });

  it('submits an absolute local path to analyze', async () => {
    render(<RepoAnalyzer variant="sheet" onComplete={vi.fn()} />);

    fireEvent.change(screen.getByLabelText('Local Folder Path'), {
      target: { value: 'C:\\repos\\GitNexus' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Analyze Repository/i }));

    await waitFor(() => {
      expect(startAnalyzeMock).toHaveBeenCalledWith({ path: 'C:\\repos\\GitNexus' });
    });
  });
});
