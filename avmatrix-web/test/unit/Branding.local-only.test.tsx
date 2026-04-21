import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const setHelpDialogBoxOpen = vi.fn();

vi.mock('../../src/components/RepoAnalyzer', () => ({
  RepoAnalyzer: () => <div data-testid="repo-analyzer">repo-analyzer</div>,
}));
vi.mock('../../src/components/EmbeddingStatus', () => ({
  EmbeddingStatus: () => <div data-testid="embedding-status">embedding-status</div>,
}));
vi.mock('../../src/hooks/useAppState.local-runtime', () => ({
  useAppState: () => ({
    projectName: 'Website',
    graph: null,
    openChatPanel: vi.fn(),
    isRightPanelOpen: false,
    rightPanelTab: 'chat',
    setSettingsPanelOpen: vi.fn(),
    setHelpDialogBoxOpen,
  }),
}));
vi.mock('../../src/services/backend-client', () => ({
  deleteRepo: vi.fn(),
  fetchRepos: vi.fn(),
  startAnalyze: vi.fn(),
  streamAnalyzeProgress: vi.fn(),
}));

import { AnalyzeOnboarding } from '../../src/components/AnalyzeOnboarding';
import { Header } from '../../src/components/Header';
import { HelpPanel } from '../../src/components/HelpPanel';
import { RepoLanding } from '../../src/components/RepoLanding';

describe('AVmatrix branding on active local surfaces', () => {
  beforeEach(() => {
    setHelpDialogBoxOpen.mockClear();
  });

  it('uses AVmatrix copy in AnalyzeOnboarding', () => {
    render(<AnalyzeOnboarding onComplete={() => {}} />);

    expect(screen.getByText('AVmatrix')).toBeInTheDocument();
    expect(screen.getByText(/AVmatrix will parse the code/i)).toBeInTheDocument();
  });

  it('uses AVmatrix copy in RepoLanding', () => {
    render(<RepoLanding repos={[]} onSelectRepo={() => {}} onAnalyzeComplete={() => {}} />);

    expect(screen.getByText('AVmatrix')).toBeInTheDocument();
    expect(screen.getByText(/Indexed locally by AVmatrix/i)).toBeInTheDocument();
  });

  it('uses AVmatrix copy in HelpPanel', () => {
    render(<HelpPanel isOpen={true} onClose={() => {}} nodeCount={10} edgeCount={20} />);

    expect(screen.getByText('What is AVmatrix?')).toBeInTheDocument();
    expect(screen.getByText('AVmatrix — graph explorer')).toBeInTheDocument();
    expect(screen.getByText('AVmatrix — local-first codebase graph explorer')).toBeInTheDocument();
  });

  it('uses AVmatrix branding in Header active path', () => {
    render(<Header availableRepos={[]} />);

    expect(screen.getByText('AVmatrix')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /about avmatrix/i })).toBeInTheDocument();
    expect(screen.queryByText('avmatrix')).not.toBeInTheDocument();
  });

  it('routes the header brand action to local help instead of upstream GitHub', () => {
    render(<Header availableRepos={[]} />);

    fireEvent.click(screen.getByRole('button', { name: /about avmatrix/i }));

    expect(setHelpDialogBoxOpen).toHaveBeenCalledWith(true);
  });
});
