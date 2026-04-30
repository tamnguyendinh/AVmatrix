import {
  Search,
  Settings,
  HelpCircle,
  FolderOpen,
  ChevronDown,
  Trash2,
  RefreshCw,
  Loader2,
} from '@/lib/lucide-icons';
import { useAppState } from '../hooks/useAppState.local-runtime';
import {
  deleteRepo,
  fetchRepos,
  startAnalyze,
  streamAnalyzeProgress,
  type BackendRepo,
  type JobProgress,
} from '../services/backend-client';
import { useState, useMemo, useRef, useEffect } from 'react';
import { GraphNode } from 'avmatrix-shared';
import { EmbeddingStatus } from './EmbeddingStatus';
import { RepoAnalyzer } from './RepoAnalyzer';

// Color mapping for node types in search results
const NODE_TYPE_COLORS: Record<string, string> = {
  Folder: '#6366f1',
  File: '#3b82f6',
  Function: '#10b981',
  Class: '#f59e0b',
  Method: '#14b8a6',
  Interface: '#ec4899',
  Variable: '#64748b',
  Import: '#475569',
  Type: '#a78bfa',
};

interface HeaderProps {
  onFocusNode?: (nodeId: string) => void;
  availableRepos?: BackendRepo[];
  openRepoAnalyzerRequestId?: number;
  onSwitchRepo?: (repoName: string) => void;
  /** Called when a newly-analyzed repo is ready; triggers connectToServer. */
  onAnalyzeComplete?: (repoName: string) => void;
  /** Called after a repo is deleted or list needs refresh. */
  onReposChanged?: (repos: BackendRepo[]) => void;
}

export const Header = ({
  onFocusNode,
  availableRepos = [],
  openRepoAnalyzerRequestId = 0,
  onSwitchRepo,
  onAnalyzeComplete,
  onReposChanged,
}: HeaderProps) => {
  const {
    projectName,
    graph,
    openChatPanel,
    isRightPanelOpen,
    rightPanelTab,
    setSettingsPanelOpen,
    setHelpDialogBoxOpen,
  } = useAppState();
  const [searchQuery, setSearchQuery] = useState('');
  const [isRepoDropdownOpen, setIsRepoDropdownOpen] = useState(false);
  const [showAnalyzer, setShowAnalyzer] = useState(false);
  const [reanalyzing, setReanalyzing] = useState<string | null>(null); // repo name being re-analyzed
  const [reanalyzeProgress, setReanalyzeProgress] = useState<JobProgress | null>(null);
  const reanalyzeSseRef = useRef<AbortController | null>(null);
  const repoDropdownRef = useRef<HTMLDivElement>(null);
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const searchRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Search results - filter nodes by name
  const searchResults = useMemo(() => {
    if (!graph || !searchQuery.trim()) return [];

    const query = searchQuery.toLowerCase();
    return graph.nodes
      .filter((node) => node.properties.name.toLowerCase().includes(query))
      .slice(0, 10); // Limit to 10 results
  }, [graph, searchQuery]);

  // Handle clicking outside search or repo dropdown to close them
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (searchRef.current && !searchRef.current.contains(e.target as Node)) {
        setIsSearchOpen(false);
      }
      if (repoDropdownRef.current && !repoDropdownRef.current.contains(e.target as Node)) {
        setIsRepoDropdownOpen(false);
        setShowAnalyzer(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Cleanup re-analyze SSE on unmount
  useEffect(() => {
    return () => {
      reanalyzeSseRef.current?.abort();
    };
  }, []);

  useEffect(() => {
    if (openRepoAnalyzerRequestId === 0) return;
    setIsRepoDropdownOpen(true);
    setShowAnalyzer(true);
  }, [openRepoAnalyzerRequestId]);

  // Keyboard shortcut (Cmd+K / Ctrl+K)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        inputRef.current?.focus();
        setIsSearchOpen(true);
      }
      if (e.key === 'Escape') {
        setIsSearchOpen(false);
        inputRef.current?.blur();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  // Handle keyboard navigation in results
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!isSearchOpen || searchResults.length === 0) return;

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelectedIndex((i) => Math.min(i + 1, searchResults.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelectedIndex((i) => Math.max(i - 1, 0));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const selected = searchResults[selectedIndex];
      if (selected) {
        handleSelectNode(selected);
      }
    }
  };

  const handleSelectNode = (node: GraphNode) => {
    // onFocusNode handles both camera focus AND selection in useSigma
    onFocusNode?.(node.id);
    setSearchQuery('');
    setIsSearchOpen(false);
    setSelectedIndex(0);
  };

  return (
    <header className="flex min-h-[72px] items-center justify-between border-b-[3px] border-border-default bg-surface px-5 py-4">
      {/* Left section */}
      <div className="flex items-center gap-4">
        <div className="press-title text-2xl leading-none text-text-primary">AVmatrix</div>
      </div>
      {/* Center - Search */}
      <div className="mr-6 ml-16 flex flex-1 items-center justify-start gap-6">
        <div className="relative w-full max-w-md" ref={searchRef}>
          <div className="press-inset flex items-center gap-2.5 px-3.5 py-2 transition-all focus-within:border-border-strong">
            <Search className="h-4 w-4 flex-shrink-0 text-text-muted" />
            <input
              ref={inputRef}
              type="text"
              placeholder="Search nodes..."
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                setIsSearchOpen(true);
                setSelectedIndex(0);
              }}
              onFocus={() => setIsSearchOpen(true)}
              onKeyDown={handleKeyDown}
              className="flex-1 border-none bg-transparent font-mono text-sm text-text-primary outline-none placeholder:text-text-muted"
            />
          </div>

          {isSearchOpen && searchQuery.trim() && (
            <div className="press-panel absolute top-full right-0 left-0 z-50 mt-2 overflow-hidden shadow-[var(--shadow-dropdown)]">
              {searchResults.length === 0 ? (
                <div className="px-4 py-3 font-reading text-sm text-text-muted">
                  No nodes found for &ldquo;{searchQuery}&rdquo;
                </div>
              ) : (
                <div className="max-h-80 overflow-y-auto">
                  {searchResults.map((node, index) => (
                    <button
                      key={node.id}
                      onClick={() => handleSelectNode(node)}
                      className={`flex w-full cursor-pointer items-center gap-3 px-4 py-2.5 text-left transition-colors ${
                        index === selectedIndex
                          ? 'bg-base text-text-primary'
                          : 'text-text-secondary hover:bg-base'
                      }`}
                    >
                      <span
                        className="h-2.5 w-2.5 flex-shrink-0 rounded-full"
                        style={{ backgroundColor: NODE_TYPE_COLORS[node.label] || '#6b7280' }}
                      />
                      <span className="flex-1 truncate text-sm font-medium">
                        {node.properties.name}
                      </span>
                      <span className="rounded border border-border-default bg-base px-2 py-0.5 font-mono text-xs text-text-secondary">
                        {node.label}
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Right section */}
      <div className="flex items-center gap-2">
        {projectName && (
          <div className="relative" ref={repoDropdownRef}>
            <button
              onClick={() => {
                setIsRepoDropdownOpen((prev) => !prev);
                setShowAnalyzer(false);
              }}
              className={`flex w-[14rem] cursor-pointer items-center gap-2 rounded-md border-[2px] px-3 py-2 font-mono text-sm transition-all ${
                isRepoDropdownOpen
                  ? 'border-border-strong bg-inset text-text-primary'
                  : 'border-border-default bg-base text-text-secondary hover:border-border-strong'
              } `}
            >
              <span className="h-1.5 w-1.5 rounded-full bg-border-strong" />
              <span className="min-w-0 flex-1 truncate">{projectName}</span>
              <ChevronDown
                className={`h-3 w-3 text-text-muted transition-transform duration-200 ${isRepoDropdownOpen ? 'rotate-180' : ''}`}
              />
            </button>

            {isRepoDropdownOpen && (
              <div className="press-panel absolute top-full left-0 z-50 mt-2 w-80 animate-slide-up overflow-hidden shadow-[var(--shadow-dropdown)]">
                {showAnalyzer ? (
                  <div className="p-4">
                    <RepoAnalyzer
                      variant="sheet"
                      onComplete={(repoName) => {
                        setShowAnalyzer(false);
                        setIsRepoDropdownOpen(false);
                        onAnalyzeComplete?.(repoName);
                      }}
                      onCancel={() => setShowAnalyzer(false)}
                    />
                  </div>
                ) : (
                  <>
                    {/* Repo list */}
                    {availableRepos.length > 0 && (
                      <div>
                        <div className="press-eyebrow px-3 pt-3 pb-2">
                          Repositories
                        </div>
                        {availableRepos.map((repo) => (
                          <div
                            key={repo.name}
                            className={`group flex items-center gap-2 px-4 py-3 transition-colors ${
                              repo.name === projectName
                                ? 'border-l-[3px] border-border-strong bg-base'
                                : 'hover:bg-base'
                            }`}
                          >
                            <button
                              onClick={() => {
                                if (repo.name !== projectName) onSwitchRepo?.(repo.name);
                                setIsRepoDropdownOpen(false);
                              }}
                              className="flex min-w-0 flex-1 cursor-pointer items-center gap-3 text-left"
                            >
                              <FolderOpen className="h-3.5 w-3.5 shrink-0 text-border-strong" />
                              <span className="flex-1 truncate font-mono text-sm text-text-primary">
                                {repo.name}
                              </span>
                              {repo.name === projectName && (
                                <span className="press-badge shrink-0 border-border-strong bg-base px-1.5 py-0.5 text-[10px] text-text-primary normal-case tracking-normal">
                                  active
                                </span>
                              )}
                            </button>
                            {/* Re-analyze */}
                            <button
                              onClick={async (e) => {
                                e.stopPropagation();
                                if (reanalyzing) return; // already running
                                setReanalyzing(repo.name);
                                setReanalyzeProgress({
                                  phase: 'queued',
                                  percent: 0,
                                  message: 'Starting...',
                                });
                                try {
                                  const { jobId } = await startAnalyze({
                                    path: repo.path,
                                    force: true,
                                  });
                                  reanalyzeSseRef.current = streamAnalyzeProgress(
                                    jobId,
                                    (p) => setReanalyzeProgress(p),
                                    () => {
                                      setReanalyzing(null);
                                      setReanalyzeProgress(null);
                                      reanalyzeSseRef.current = null;
                                      onAnalyzeComplete?.(repo.name);
                                    },
                                    (errMsg) => {
                                      console.error('Re-analyze failed:', errMsg);
                                      setReanalyzing(null);
                                      setReanalyzeProgress(null);
                                      reanalyzeSseRef.current = null;
                                    },
                                  );
                                } catch (err) {
                                  console.error('Failed to start re-analysis:', err);
                                  setReanalyzing(null);
                                  setReanalyzeProgress(null);
                                }
                              }}
                              disabled={!!reanalyzing}
                              className={`cursor-pointer rounded p-1 transition-all ${
                                reanalyzing === repo.name
                                  ? 'text-border-strong'
                                  : 'text-text-muted/0 group-hover:text-text-muted hover:!text-border-strong'
                              }`}
                              title={
                                reanalyzing === repo.name
                                  ? 'Re-analyzing...'
                                  : `Re-analyze ${repo.name}`
                              }
                            >
                              <RefreshCw
                                className={`h-3.5 w-3.5 ${reanalyzing === repo.name ? 'animate-spin' : ''}`}
                              />
                            </button>
                            {/* Delete */}
                            <button
                              onClick={async (e) => {
                                e.stopPropagation();
                                // Abort any running re-analysis for this repo
                                if (reanalyzing === repo.name) {
                                  reanalyzeSseRef.current?.abort();
                                  setReanalyzing(null);
                                  setReanalyzeProgress(null);
                                  reanalyzeSseRef.current = null;
                                }
                                try {
                                  await deleteRepo(repo.name);
                                  const updated = await fetchRepos();
                                  onReposChanged?.(updated);
                                  // If we deleted the active repo, switch to first available
                                  if (repo.name === projectName && updated.length > 0) {
                                    onSwitchRepo?.(updated[0].name);
                                  } else if (updated.length === 0) {
                                    // No repos left — go back to onboarding
                                    window.location.reload();
                                  }
                                } catch (err) {
                                  console.error('Failed to delete repo:', err);
                                }
                              }}
                              className="cursor-pointer rounded p-1 text-text-muted/0 transition-all group-hover:text-text-muted hover:!text-red-400"
                              title={`Delete ${repo.name}`}
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          </div>
                        ))}
                      </div>
                    )}

                    {/* Re-analyze progress bar */}
                    {reanalyzing && reanalyzeProgress && (
                      <div className="border-t border-border-subtle bg-base px-4 py-2.5">
                        <div className="mb-1.5 flex items-center gap-2">
                          <Loader2 className="h-3 w-3 shrink-0 animate-spin text-border-strong" />
                          <span className="truncate text-xs text-text-secondary">
                            Re-analyzing {reanalyzing}: {reanalyzeProgress.message}
                          </span>
                        </div>
                        <div className="h-1 overflow-hidden rounded-full bg-inset">
                          <div
                            className="h-full rounded-full bg-border-strong transition-all duration-300"
                            style={{ width: `${Math.max(2, reanalyzeProgress.percent)}%` }}
                          />
                        </div>
                      </div>
                    )}

                    {/* Analyze new */}
                    <div
                      className={
                        availableRepos.length > 0 || reanalyzing
                          ? 'border-t border-border-subtle'
                          : ''
                      }
                    >
                      <button
                        onClick={() => setShowAnalyzer(true)}
                        disabled={!!reanalyzing}
                        className="flex w-full cursor-pointer items-center px-4 py-3 text-left transition-colors hover:bg-base disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        <span className="font-reading text-sm text-text-secondary">
                          Analyze a new repository...
                        </span>
                      </button>
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
        )}

        <EmbeddingStatus />

        <button
          onClick={() => setSettingsPanelOpen(true)}
          className="press-ghost-button flex h-9 w-9 cursor-pointer items-center justify-center text-text-secondary"
          title="Session Settings"
        >
          <Settings className="h-4.5 w-4.5" />
        </button>
        <button
          title="Help"
          onClick={() => setHelpDialogBoxOpen(true)}
          className="press-ghost-button flex h-9 w-9 cursor-pointer items-center justify-center text-text-secondary"
        >
          <HelpCircle className="h-4.5 w-4.5" />
        </button>

        <button
          onClick={openChatPanel}
          className={`press-filled-button flex items-center gap-1.5 px-3.5 py-2 text-sm font-medium ${
            isRightPanelOpen && rightPanelTab === 'chat'
              ? 'bg-accent-dim text-text-inverse'
              : 'text-text-inverse'
          } `}
        >
          <span>My AI</span>
        </button>
      </div>
    </header>
  );
};
