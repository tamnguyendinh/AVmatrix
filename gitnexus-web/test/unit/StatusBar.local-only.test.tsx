import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockUseAppState = vi.fn();

vi.mock('../../src/hooks/useAppState.local-runtime', () => ({
  useAppState: () => mockUseAppState(),
}));

import { StatusBar } from '../../src/components/StatusBar';

describe('StatusBar.local-only', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders local status and graph stats without sponsor or SWE-bench CTA', () => {
    mockUseAppState.mockReturnValue({
      progress: null,
      graph: {
        nodes: [
          { properties: { language: 'TypeScript' } },
          { properties: { language: 'TypeScript' } },
          { properties: { language: 'Python' } },
        ],
        relationships: [{}, {}],
      },
    });

    render(<StatusBar />);

    expect(screen.getByTestId('status-ready')).toHaveTextContent('Ready');
    expect(screen.getByTestId('graph-stats')).toHaveTextContent('3 nodes');
    expect(screen.getByTestId('graph-stats')).toHaveTextContent('2 edges');
    expect(screen.getByTestId('graph-stats')).toHaveTextContent('TypeScript');
    expect(screen.queryByText(/Sponsor/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/SWE-bench/i)).not.toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('keeps progress rendering when an active job is running', () => {
    mockUseAppState.mockReturnValue({
      progress: {
        phase: 'extracting',
        percent: 42,
        message: 'Analyzing local repository...',
      },
      graph: null,
    });

    render(<StatusBar />);

    expect(screen.queryByTestId('status-ready')).not.toBeInTheDocument();
    expect(screen.getByText('Analyzing local repository...')).toBeInTheDocument();
    expect(screen.queryByText(/Sponsor/i)).not.toBeInTheDocument();
  });
});
