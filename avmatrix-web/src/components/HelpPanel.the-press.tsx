import React, { useState } from 'react';
import {
  X,
  GitBranch,
  Search,
  Zap,
  Keyboard,
  BarChart2,
  HelpCircle,
} from 'lucide-react';

interface HelpPanelProps {
  isOpen: boolean;
  onClose: () => void;
  nodeCount: number;
  edgeCount: number;
}

type TabId = 'overview' | 'graph' | 'search' | 'ai' | 'shortcuts' | 'status';

interface Tab {
  id: TabId;
  label: string;
  icon: React.ReactNode;
}

const tabs: Tab[] = [
  { id: 'overview', label: 'Overview', icon: <HelpCircle className="h-4 w-4" /> },
  { id: 'graph', label: 'Graph & nodes', icon: <GitBranch className="h-4 w-4" /> },
  { id: 'search', label: 'Search & filter', icon: <Search className="h-4 w-4" /> },
  { id: 'ai', label: 'Desk Chat', icon: <Zap className="h-4 w-4" /> },
  { id: 'shortcuts', label: 'Shortcuts', icon: <Keyboard className="h-4 w-4" /> },
  { id: 'status', label: 'Status bar', icon: <BarChart2 className="h-4 w-4" /> },
];

const shortcuts = [
  { label: 'Search nodes', mac: '⌘ K', win: 'Ctrl K' },
  { label: 'Deselect / close', mac: 'Esc', win: 'Esc' },
];

const nodeColors = [
  { color: '#10b981', label: 'Function', desc: 'Function declarations' },
  { color: '#4f86c6', label: 'File', desc: 'Source files' },
  { color: '#b77b3f', label: 'Class', desc: 'Class declarations' },
  { color: '#4d7b77', label: 'Method', desc: 'Class methods' },
  { color: '#8f5873', label: 'Interface', desc: 'TypeScript interfaces' },
  { color: '#7d7aa3', label: 'Folder', desc: 'Directory nodes' },
];

const statusItems = (nodeCount: number, edgeCount: number) => [
  {
    badge: <span className="h-2.5 w-2.5 rounded-full bg-success" />,
    title: 'Ready',
    desc: 'Graph is fully loaded and interactive.',
  },
  {
    badge: <span className="font-mono text-xs text-border-strong">{nodeCount}</span>,
    title: 'Nodes count',
    desc: 'Total files and symbols in the graph.',
  },
  {
    badge: <span className="font-mono text-xs text-info">{edgeCount}</span>,
    title: 'Edges count',
    desc: 'Import and dependency connections.',
  },
  {
    badge: <span className="font-mono text-[11px] text-success">Semantic Ready</span>,
    title: 'AI index status',
    desc: 'Repo is indexed for local semantic queries.',
  },
];

const Kbd = ({
  children,
  variant = 'default',
}: {
  children: React.ReactNode;
  variant?: 'default' | 'windows';
}) => (
  <kbd
    className={`rounded border px-2 py-0.5 font-mono text-[11px] ${
      variant === 'windows'
        ? 'border-info/40 bg-base text-info'
        : 'border-border-subtle bg-base text-text-primary'
    }`}
  >
    {children}
  </kbd>
);

const InfoCard = ({
  title,
  children,
  tone = 'default',
}: {
  title: string;
  children: React.ReactNode;
  tone?: 'default' | 'strong' | 'success' | 'warning' | 'info';
}) => {
  const toneClass =
    tone === 'strong'
      ? 'border-border-strong'
      : tone === 'success'
        ? 'border-success'
        : tone === 'warning'
          ? 'border-warning'
          : tone === 'info'
            ? 'border-info'
            : 'border-border-default';

  return (
    <div className={`rounded-xl border-[2px] ${toneClass} bg-base p-4`}>
      <p className="press-eyebrow mb-2 text-text-secondary">{title}</p>
      <div className="font-reading text-sm leading-relaxed text-text-secondary">{children}</div>
    </div>
  );
};

function TabContent({
  active,
  nodeCount,
  edgeCount,
}: {
  active: TabId;
  nodeCount: number;
  edgeCount: number;
}) {
  if (active === 'overview') {
    return (
      <div className="space-y-4">
        <p className="press-eyebrow text-text-secondary">Getting started</p>

        <InfoCard title="What is AVmatrix?" tone="strong">
          A local-first graph explorer for your codebase. Files, functions, imports, and
          relationships become a navigable graph you can inspect visually.
        </InfoCard>

        <InfoCard title="Your current repo" tone="info">
          Loaded graph: <span className="font-mono text-text-primary">{nodeCount}</span> nodes and{' '}
          <span className="font-mono text-text-primary">{edgeCount}</span> edges.
        </InfoCard>

        <InfoCard title="Three ways to explore" tone="default">
          <div className="space-y-1">
            <p>1. Click nodes to inspect them.</p>
            <p>2. Search by name or type.</p>
            <p>3. Ask Desk Chat a natural-language question.</p>
          </div>
        </InfoCard>

        <InfoCard title="Navigation" tone="warning">
          <div className="space-y-1">
            <p>Scroll to zoom.</p>
            <p>Click and drag to pan.</p>
            <p>Double-click a node to focus its subgraph.</p>
          </div>
        </InfoCard>
      </div>
    );
  }

  if (active === 'graph') {
    return (
      <div className="space-y-4">
        <p className="press-eyebrow text-text-secondary">Node color legend</p>

        <div className="space-y-3">
          {nodeColors.map(({ color, label, desc }) => (
            <div
              key={label}
              className="flex items-start gap-3 rounded-xl border-[2px] border-border-default bg-base p-3"
            >
              <span className="mt-1 h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: color }} />
              <div>
                <p className="font-mono text-sm text-text-primary">{label} nodes</p>
                <p className="font-reading text-sm text-text-secondary">{desc}</p>
              </div>
            </div>
          ))}
        </div>

        <div className="press-divider border-t pt-4">
          <p className="font-reading text-sm leading-relaxed text-text-secondary">
            Node <strong className="text-text-primary">size</strong> reflects connection count.
            Larger nodes are depended on by more files, and edges point from importer to imported.
          </p>
        </div>

        <InfoCard title="Detail panel" tone="default">
          Click any node to open its detail panel and inspect imports, exports, and reverse
          dependencies.
        </InfoCard>
      </div>
    );
  }

  if (active === 'search') {
    return (
      <div className="space-y-4">
        <p className="press-eyebrow text-text-secondary">Search & filter</p>

        <InfoCard title="Search nodes" tone="strong">
          <div className="mb-2 flex items-center gap-2">
            <Kbd>⌘ K</Kbd>
            <span className="font-mono text-xs text-text-secondary">/</span>
            <Kbd>Ctrl K</Kbd>
          </div>
          Search by filename, function name, or import path. Matching nodes are highlighted live
          in the graph.
        </InfoCard>

        <InfoCard title="Filter panel" tone="default">
          Use the filter panel in the left rail to isolate node types, hide noise, or limit the
          graph to a depth range around a selected node.
        </InfoCard>

        <div className="rounded-xl border-[2px] border-border-default bg-base p-4">
          <p className="press-eyebrow mb-3 text-text-secondary">Search syntax</p>
          <div className="space-y-2">
            {[
              { query: 'auth', hint: 'match by name fragment' },
              { query: './utils/', hint: 'match by path prefix' },
              { query: 'type:config', hint: 'filter by node type' },
            ].map(({ query, hint }) => (
              <div key={query} className="flex items-baseline gap-3">
                <code className="rounded border border-border-subtle bg-surface px-2 py-0.5 font-mono text-[11px] text-text-primary">
                  {query}
                </code>
                <span className="font-reading text-sm text-text-secondary">{hint}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (active === 'ai') {
    return (
      <div className="space-y-4">
        <p className="press-eyebrow text-text-secondary">Desk Chat</p>

        <InfoCard title="Semantic ready" tone="success">
          Your repo is indexed and ready for semantic queries. Desk Chat runs through the local
          session runtime, so it understands code structure and relationships instead of only file
          names.
        </InfoCard>

        <div>
          <p className="mb-2 font-mono text-xs uppercase tracking-[0.16em] text-text-secondary">
            Try asking
          </p>
          <div className="space-y-2">
            {[
              'Which files depend on the auth module?',
              'Find circular dependencies in this repo.',
              'What are the most connected components?',
              'Show me all files that import useEffect.',
            ].map((q) => (
              <div
                key={q}
                className="rounded-xl border-[2px] border-border-default bg-base px-4 py-3 font-reading text-sm italic text-text-primary"
              >
                “{q}”
              </div>
            ))}
          </div>
        </div>

        <p className="font-reading text-sm leading-relaxed text-text-secondary">
          Open the prompt from the <span className="font-mono text-text-primary">Desk Chat</span>{' '}
          button in the top bar.
        </p>
      </div>
    );
  }

  if (active === 'shortcuts') {
    return (
      <div className="overflow-hidden rounded-xl border-[2px] border-border-default bg-base">
        <div className="grid grid-cols-[1fr_84px_92px] gap-2 border-b border-border-subtle px-4 py-3">
          <span className="press-eyebrow text-text-secondary">Action</span>
          <span className="press-eyebrow text-center text-text-secondary">Mac</span>
          <span className="press-eyebrow text-center text-text-secondary">Windows</span>
        </div>
        {shortcuts.map(({ label, mac, win }, index) => (
          <div
            key={label}
            className={`grid grid-cols-[1fr_84px_92px] gap-2 px-4 py-3 ${
              index < shortcuts.length - 1 ? 'border-b border-border-subtle' : ''
            }`}
          >
            <span className="font-reading text-sm text-text-secondary">{label}</span>
            <span className="flex justify-center">
              <Kbd>{mac}</Kbd>
            </span>
            <span className="flex justify-center">
              <Kbd variant="windows">{win}</Kbd>
            </span>
          </div>
        ))}
      </div>
    );
  }

  if (active === 'status') {
    return (
      <div className="space-y-3">
        <p className="press-eyebrow text-text-secondary">Status bar explained</p>
        {statusItems(nodeCount, edgeCount).map(({ badge, title, desc }) => (
          <div
            key={title}
            className="flex items-center gap-3 rounded-xl border-[2px] border-border-default bg-base p-4"
          >
            <div className="flex min-w-8 items-center justify-center">{badge}</div>
            <div>
              <p className="font-mono text-sm text-text-primary">{title}</p>
              <p className="font-reading text-sm text-text-secondary">{desc}</p>
            </div>
          </div>
        ))}
      </div>
    );
  }

  return null;
}

export const HelpPanelThePress = ({ isOpen, onClose, nodeCount, edgeCount }: HelpPanelProps) => {
  const [active, setActive] = useState<TabId>('overview');

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />

      <div className="press-panel relative mx-4 flex h-[min(78vh,760px)] w-full max-w-5xl flex-col overflow-hidden shadow-[var(--shadow-dropdown)]">
        <div className="flex items-center justify-between border-b-[3px] border-border-default bg-base px-6 py-5">
          <div className="flex items-center gap-4">
            <div className="flex h-11 w-11 items-center justify-center rounded-xl border-[3px] border-border-strong bg-inset">
              <HelpCircle className="h-5 w-5 text-border-strong" />
            </div>
            <div>
              <p className="press-eyebrow">Reference desk</p>
              <h2 className="press-title text-2xl">Help & Reference</h2>
              <p className="font-reading text-sm text-text-secondary">
                Local guide for the AVmatrix browser UI and workspace graph.
              </p>
            </div>
          </div>
          <button onClick={onClose} className="press-ghost-button rounded-lg p-2 text-text-secondary">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="grid min-h-0 flex-1 grid-cols-[220px_1fr]">
          <div className="border-r-[3px] border-border-default bg-surface p-3">
            <div className="space-y-1">
              {tabs.map(({ id, label, icon }) => {
                const isActive = active === id;
                return (
                  <button
                    key={id}
                    onClick={() => setActive(id)}
                    className={`flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors ${
                      isActive
                        ? 'border-[2px] border-border-strong bg-base text-text-primary'
                        : 'border-[2px] border-transparent text-text-secondary hover:border-border-default hover:bg-base hover:text-text-primary'
                    }`}
                  >
                    <span className={isActive ? 'text-border-strong' : 'text-text-muted'}>{icon}</span>
                    <span className="font-mono text-xs">{label}</span>
                  </button>
                );
              })}
            </div>
          </div>

          <div className="scrollbar-thin min-h-0 overflow-y-auto bg-base p-6">
            <TabContent active={active} nodeCount={nodeCount} edgeCount={edgeCount} />
          </div>
        </div>

        <div className="flex items-center justify-between border-t-[3px] border-border-default bg-surface px-6 py-3">
          <span className="font-mono text-[11px] uppercase tracking-[0.12em] text-text-secondary">
            AVmatrix · local-first graph explorer
          </span>
          <span className="font-reading text-xs text-text-secondary">
            Browser UI with a local runtime bridge
          </span>
        </div>
      </div>
    </div>
  );
};
