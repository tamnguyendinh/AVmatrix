import { useState, useRef, useEffect, useCallback } from 'react';
import { Terminal, Play, X, ChevronDown, ChevronUp, Loader2, Table } from '@/lib/lucide-icons';
import { useAppState } from '../hooks/useAppState.local-runtime';

const EXAMPLE_QUERIES = [
  {
    label: 'All Functions',
    query: `MATCH (n:Function) RETURN n.id AS id, n.name AS name, n.filePath AS path LIMIT 50`,
  },
  {
    label: 'All Classes',
    query: `MATCH (n:Class) RETURN n.id AS id, n.name AS name, n.filePath AS path LIMIT 50`,
  },
  {
    label: 'All Interfaces',
    query: `MATCH (n:Interface) RETURN n.id AS id, n.name AS name, n.filePath AS path LIMIT 50`,
  },
  {
    label: 'Function Calls',
    query: `MATCH (a:File)-[r:CodeRelation {type: 'CALLS'}]->(b:Function) RETURN a.id AS id, a.name AS caller, b.name AS callee LIMIT 50`,
  },
  {
    label: 'Import Dependencies',
    query: `MATCH (a:File)-[r:CodeRelation {type: 'IMPORTS'}]->(b:File) RETURN a.id AS id, a.name AS from, b.name AS imports LIMIT 50`,
  },
];

export const QueryFAB = () => {
  const {
    setHighlightedNodeIds,
    setQueryResult,
    queryResult,
    clearQueryHighlights,
    graph,
    runQuery,
    isDatabaseReady,
  } = useAppState();

  const [isExpanded, setIsExpanded] = useState(false);
  const [query, setQuery] = useState('');
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showExamples, setShowExamples] = useState(false);
  const [showResults, setShowResults] = useState(true);

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isExpanded && textareaRef.current) {
      textareaRef.current.focus();
    }
  }, [isExpanded]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setShowExamples(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isExpanded) {
        setIsExpanded(false);
        setShowExamples(false);
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isExpanded]);

  const handleRunQuery = useCallback(async () => {
    if (!query.trim() || isRunning) return;

    if (!graph) {
      setError('No project loaded. Load a project first.');
      return;
    }

    const ready = await isDatabaseReady();
    if (!ready) {
      setError('Database not ready. Please wait for loading to complete.');
      return;
    }

    setIsRunning(true);
    setError(null);

    const startTime = performance.now();

    try {
      const rows = await runQuery(query);
      const executionTime = performance.now() - startTime;

      // Extract node IDs from results - handles various formats
      // 1. Array format: first element if it looks like a node ID
      // 2. Object format: any field ending with 'id' (case-insensitive)
      // 3. Values matching node ID pattern: Label:path:name
      const nodeIdPattern = /^(File|Function|Class|Method|Interface|Folder|CodeElement):/;

      const nodeIds = rows
        .flatMap((row) => {
          const ids: string[] = [];

          if (Array.isArray(row)) {
            // Array format - check all elements for node ID patterns
            row.forEach((val) => {
              if (typeof val === 'string' && (nodeIdPattern.test(val) || val.includes(':'))) {
                ids.push(val);
              }
            });
          } else if (typeof row === 'object' && row !== null) {
            // Object format - check fields ending with 'id' and values matching patterns
            Object.entries(row).forEach(([key, val]) => {
              const keyLower = key.toLowerCase();
              if (typeof val === 'string') {
                // Field name contains 'id'
                if (keyLower.includes('id') || keyLower === 'id') {
                  ids.push(val);
                }
                // Value matches node ID pattern
                else if (nodeIdPattern.test(val)) {
                  ids.push(val);
                }
              }
            });
          }

          return ids;
        })
        .filter(Boolean)
        .filter((id, index, arr) => arr.indexOf(id) === index);

      setQueryResult({ rows, nodeIds, executionTime });
      setHighlightedNodeIds(new Set(nodeIds));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Query execution failed');
      setQueryResult(null);
      setHighlightedNodeIds(new Set());
    } finally {
      setIsRunning(false);
    }
  }, [query, isRunning, graph, isDatabaseReady, runQuery, setHighlightedNodeIds, setQueryResult]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleRunQuery();
    }
  };

  const handleSelectExample = (exampleQuery: string) => {
    setQuery(exampleQuery);
    setShowExamples(false);
    textareaRef.current?.focus();
  };

  const handleClose = () => {
    setIsExpanded(false);
    setShowExamples(false);
    clearQueryHighlights();
    setError(null);
  };

  const handleClear = () => {
    setQuery('');
    clearQueryHighlights();
    setError(null);
    textareaRef.current?.focus();
  };

  if (!isExpanded) {
    return (
      <button
        onClick={() => setIsExpanded(true)}
        className="workspace-outline-button group absolute bottom-4 left-4 z-20 flex items-center gap-2 rounded-xl bg-workspace-surface px-4 py-2.5 font-mono text-sm font-medium text-workspace-text-primary transition-all duration-200 hover:-translate-y-0.5"
      >
        <Terminal className="h-4 w-4" />
        <span>Query</span>
        {queryResult && queryResult.nodeIds.length > 0 && (
          <span className="ml-1 rounded-md border border-workspace-border-default bg-workspace-base px-1.5 py-0.5 text-xs font-semibold">
            {queryResult.nodeIds.length}
          </span>
        )}
      </button>
    );
  }

  return (
    <div
      ref={panelRef}
      className="workspace-panel absolute bottom-4 left-4 z-20 w-[480px] max-w-[calc(100%-2rem)] animate-fade-in backdrop-blur-md"
    >
      <div className="flex items-center justify-between border-b-[3px] border-workspace-border-default px-4 py-3">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg border-[2px] border-workspace-border-default bg-workspace-base">
            <Terminal className="h-4 w-4 text-workspace-text-primary" />
          </div>
          <span className="font-mono text-sm font-medium text-workspace-text-primary">
            Cypher Query
          </span>
        </div>
        <button
          onClick={handleClose}
          className="workspace-outline-button rounded-md p-1.5 text-workspace-text-secondary hover:text-workspace-text-primary"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="p-3">
        <div className="relative">
          <textarea
            ref={textareaRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="MATCH (n:Function) RETURN n.name, n.filePath LIMIT 10"
            rows={3}
            className="w-full resize-none rounded-lg border-[2px] border-workspace-border-default bg-workspace-inset px-3 py-2.5 font-mono text-sm text-workspace-text-primary transition-all outline-none placeholder:text-workspace-text-muted focus:border-workspace-border-strong"
          />
        </div>

        <div className="mt-3 flex items-center justify-between">
          <div className="relative">
            <button
              onClick={() => setShowExamples(!showExamples)}
              className="workspace-outline-button flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs text-workspace-text-secondary hover:text-workspace-text-primary"
            >
              <span>Examples</span>
              <ChevronDown
                className={`h-3.5 w-3.5 transition-transform ${showExamples ? 'rotate-180' : ''}`}
              />
            </button>

            {showExamples && (
              <div className="workspace-panel absolute bottom-full left-0 mb-2 w-64 animate-fade-in py-1 shadow-[var(--shadow-dropdown)]">
                {EXAMPLE_QUERIES.map((example) => (
                  <button
                    key={example.label}
                    onClick={() => handleSelectExample(example.query)}
                    className="w-full px-3 py-2 text-left font-mono text-sm text-workspace-text-secondary transition-colors hover:bg-workspace-base hover:text-workspace-text-primary"
                  >
                    {example.label}
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="flex items-center gap-2">
            {query && (
              <button
                onClick={handleClear}
                className="workspace-outline-button rounded-md px-3 py-1.5 text-xs text-workspace-text-secondary hover:text-workspace-text-primary"
              >
                Clear
              </button>
            )}
            <button
              onClick={handleRunQuery}
              disabled={!query.trim() || isRunning}
              className="workspace-outline-button flex items-center gap-1.5 rounded-md border-workspace-border-strong bg-workspace-surface px-4 py-1.5 font-mono text-sm font-medium text-workspace-text-primary transition-all disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none"
            >
              {isRunning ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Play className="h-3.5 w-3.5" />
              )}
              <span>Run</span>
            </button>
          </div>
        </div>
      </div>

      {error && (
        <div className="border-t border-error bg-workspace-base px-4 py-2">
          <p className="font-mono text-xs text-error">{error}</p>
        </div>
      )}

      {queryResult && !error && (
        <div className="border-t border-workspace-border-default">
          <div className="flex items-center justify-between bg-workspace-base px-4 py-2.5">
            <div className="flex items-center gap-3 font-mono text-xs">
              <span className="text-workspace-text-secondary">
                <span className="font-semibold text-workspace-text-primary">
                  {queryResult.rows.length}
                </span>{' '}
                rows
              </span>
              {queryResult.nodeIds.length > 0 && (
                <span className="text-workspace-text-secondary">
                  <span className="font-semibold text-workspace-text-primary">
                    {queryResult.nodeIds.length}
                  </span>{' '}
                  highlighted
                </span>
              )}
              <span className="text-workspace-text-muted">
                {queryResult.executionTime.toFixed(1)}ms
              </span>
            </div>
            <div className="flex items-center gap-2">
              {queryResult.nodeIds.length > 0 && (
                <button
                  onClick={clearQueryHighlights}
                  className="font-mono text-xs text-workspace-text-muted transition-colors hover:text-workspace-text-primary"
                >
                  Clear
                </button>
              )}
              <button
                onClick={() => setShowResults(!showResults)}
                className="flex items-center gap-1 font-mono text-xs text-workspace-text-muted transition-colors hover:text-workspace-text-primary"
              >
                <Table className="h-3 w-3" />
                {showResults ? (
                  <ChevronDown className="h-3 w-3" />
                ) : (
                  <ChevronUp className="h-3 w-3" />
                )}
              </button>
            </div>
          </div>

          {showResults && queryResult.rows.length > 0 && (
            <div className="scrollbar-thin max-h-48 overflow-auto border-t border-workspace-border-subtle">
              <table className="w-full font-mono text-xs">
                <thead className="sticky top-0 bg-workspace-surface">
                  <tr>
                    {Object.keys(queryResult.rows[0]).map((key) => (
                      <th
                        key={key}
                        className="border-b border-workspace-border-subtle px-3 py-2 text-left font-medium text-workspace-text-muted"
                      >
                        {key}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {queryResult.rows.slice(0, 50).map((row, i) => (
                    <tr key={i} className="transition-colors hover:bg-workspace-base/50">
                      {Object.values(row).map((val, j) => (
                        <td
                          key={j}
                          className="max-w-[200px] truncate border-b border-workspace-border-subtle/50 px-3 py-1.5 font-mono text-workspace-text-secondary"
                        >
                          {typeof val === 'object' ? JSON.stringify(val) : String(val ?? '')}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
              {queryResult.rows.length > 50 && (
                <div className="border-t border-workspace-border-subtle bg-workspace-surface px-3 py-2 font-mono text-xs text-workspace-text-muted">
                  Showing 50 of {queryResult.rows.length} rows
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
