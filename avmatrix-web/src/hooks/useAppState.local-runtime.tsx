import {
  createContext,
  useContext,
  useState,
  useCallback,
  useRef,
  useEffect,
  useMemo,
  ReactNode,
  startTransition,
} from 'react';
import type { GraphNode, NodeLabel, PipelineProgress } from 'avmatrix-shared';
import type { KnowledgeGraph } from '../core/graph/types';
import { createKnowledgeGraph } from '../core/graph/graph';
import { type EdgeType } from '../lib/constants';
import {
  connectToServer,
  runQuery as backendRunQuery,
  search as backendSearch,
  grep as backendGrep,
  readFile as backendReadFile,
  startEmbeddings as backendStartEmbeddings,
  streamEmbeddingProgress,
  probeBackend,
  type BackendRepo,
  type ConnectResult,
  type JobProgress,
} from '../services/backend-client';
import { ERROR_RESET_DELAY_MS } from '../config/ui-constants';
import { normalizePath } from '../lib/path-resolution';
import { FILE_REF_REGEX, NODE_REF_REGEX } from '../lib/grounding-patterns';
import { GraphStateProvider, useGraphState } from './app-state/graph';
import type { ChatRuntimeBridge } from './chat-runtime/types';

export type ViewMode = 'onboarding' | 'loading' | 'exploring';
export type RightPanelTab = 'chat' | 'processes';
export type EmbeddingStatus = 'idle' | 'loading' | 'embedding' | 'indexing' | 'ready' | 'error';

export interface QueryResult {
  rows: Record<string, any>[];
  nodeIds: string[];
  executionTime: number;
}

// Animation types for graph nodes
export type AnimationType = 'pulse' | 'ripple' | 'glow';

export interface NodeAnimation {
  type: AnimationType;
  startTime: number;
  duration: number;
}

// Code reference from AI grounding or user selection
export interface CodeReference {
  id: string;
  filePath: string;
  startLine?: number;
  endLine?: number;
  nodeId?: string; // Associated graph node ID
  label?: string; // File, Function, Class, etc.
  name?: string; // Display name
  source: 'ai' | 'user'; // How it was added
}

export interface CodeReferenceFocus {
  filePath: string;
  startLine?: number;
  endLine?: number;
  ts: number;
}

interface AppState {
  // View state
  viewMode: ViewMode;
  setViewMode: (mode: ViewMode) => void;

  // Graph data
  graph: KnowledgeGraph | null;
  setGraph: (graph: KnowledgeGraph | null) => void;

  // Selection
  selectedNode: GraphNode | null;
  setSelectedNode: (node: GraphNode | null) => void;

  // Right Panel (unified Code + Chat)
  isRightPanelOpen: boolean;
  setRightPanelOpen: (open: boolean) => void;
  rightPanelTab: RightPanelTab;
  setRightPanelTab: (tab: RightPanelTab) => void;
  openCodePanel: () => void;
  openChatPanel: () => void;
  helpDialogBoxOpen: boolean;
  setHelpDialogBoxOpen: (open: boolean) => void;

  // Filters
  visibleLabels: NodeLabel[];
  toggleLabelVisibility: (label: NodeLabel) => void;
  visibleEdgeTypes: EdgeType[];
  toggleEdgeVisibility: (edgeType: EdgeType) => void;
  areGraphLinksVisible: boolean;
  setGraphLinksVisible: (visible: boolean) => void;
  toggleGraphLinksVisible: () => void;

  // Depth filter (N hops from selection)
  depthFilter: number | null;
  setDepthFilter: (depth: number | null) => void;

  // Query state
  highlightedNodeIds: Set<string>;
  setHighlightedNodeIds: (ids: Set<string>) => void;
  // AI highlights (toggable)
  aiCitationHighlightedNodeIds: Set<string>;
  aiToolHighlightedNodeIds: Set<string>;
  blastRadiusNodeIds: Set<string>;
  isAIHighlightsEnabled: boolean;
  toggleAIHighlights: () => void;
  clearAIToolHighlights: () => void;
  clearAICitationHighlights: () => void;
  clearBlastRadius: () => void;
  queryResult: QueryResult | null;
  setQueryResult: (result: QueryResult | null) => void;
  clearQueryHighlights: () => void;

  // Node animations (for MCP tool visual feedback)
  animatedNodes: Map<string, NodeAnimation>;
  triggerNodeAnimation: (nodeIds: string[], type: AnimationType) => void;
  clearAnimations: () => void;

  // Progress
  progress: PipelineProgress | null;
  setProgress: (progress: PipelineProgress | null) => void;

  // Project info
  projectName: string;
  setProjectName: (name: string) => void;

  // Multi-repo switching
  serverBaseUrl: string | null;
  setServerBaseUrl: (url: string | null) => void;
  availableRepos: BackendRepo[];
  setAvailableRepos: (repos: BackendRepo[]) => void;
  switchRepo: (repoName: string) => Promise<void>;
  setCurrentRepo: (repoName: string) => void;
  repoAnalyzerRequestId: number;
  requestRepoAnalyzeDialog: () => void;

  // Worker API (shared across app)
  runQuery: (cypher: string) => Promise<any[]>;
  isDatabaseReady: () => Promise<boolean>;

  // Embedding state
  embeddingStatus: EmbeddingStatus;
  embeddingProgress: { phase: string; percent: number } | null;

  // Embedding methods
  startEmbeddings: () => Promise<void>;
  startEmbeddingsWithFallback: () => void;
  semanticSearch: (query: string, k?: number) => Promise<any[]>;
  semanticSearchWithContext: (query: string, k?: number, hops?: number) => Promise<any[]>;
  isEmbeddingReady: boolean;

  isSettingsPanelOpen: boolean;
  setSettingsPanelOpen: (open: boolean) => void;

  // Code References Panel
  codeReferences: CodeReference[];
  isCodePanelOpen: boolean;
  setCodePanelOpen: (open: boolean) => void;
  addCodeReference: (ref: Omit<CodeReference, 'id'>) => void;
  removeCodeReference: (id: string) => void;
  clearAICodeReferences: () => void;
  clearCodeReferences: () => void;
  codeReferenceFocus: CodeReferenceFocus | null;
  chatRuntimeBridge: ChatRuntimeBridge;
}

const AppStateContext = createContext<AppState | null>(null);

export const AppStateProvider = ({ children }: { children: ReactNode }) => (
  <GraphStateProvider>
    <AppStateProviderInner>{children}</AppStateProviderInner>
  </GraphStateProvider>
);

const AppStateProviderInner = ({ children }: { children: ReactNode }) => {
  // View state
  const [viewMode, setViewMode] = useState<ViewMode>('onboarding');

  const {
    graph,
    setGraph,
    selectedNode,
    setSelectedNode,
    visibleLabels,
    toggleLabelVisibility,
    visibleEdgeTypes,
    toggleEdgeVisibility,
    areGraphLinksVisible,
    setGraphLinksVisible,
    toggleGraphLinksVisible,
    depthFilter,
    setDepthFilter,
    highlightedNodeIds,
    setHighlightedNodeIds,
  } = useGraphState();

  // Right Panel
  const [isRightPanelOpen, setRightPanelOpen] = useState(false);
  const [rightPanelTab, setRightPanelTab] = useState<RightPanelTab>('chat');
  const [helpDialogBoxOpen, setHelpDialogBoxOpen] = useState(false);

  const openCodePanel = useCallback(() => {
    // Legacy API: used by graph/tree selection.
    // Code is now shown in the Code References Panel (left of the graph),
    // so "openCodePanel" just ensures that panel becomes visible when needed.
    setCodePanelOpen(true);
  }, []);

  const openChatPanel = useCallback(() => {
    setRightPanelOpen(true);
    setRightPanelTab('chat');
  }, []);

  // Query state
  const [queryResult, setQueryResult] = useState<QueryResult | null>(null);

  // AI highlights (separate from user/query highlights)
  const [aiCitationHighlightedNodeIds, setAICitationHighlightedNodeIds] = useState<Set<string>>(
    new Set(),
  );
  const [aiToolHighlightedNodeIds, setAIToolHighlightedNodeIds] = useState<Set<string>>(new Set());
  const [blastRadiusNodeIds, setBlastRadiusNodeIds] = useState<Set<string>>(new Set());
  const [isAIHighlightsEnabled, setAIHighlightsEnabled] = useState(true);

  const toggleAIHighlights = useCallback(() => {
    setAIHighlightsEnabled((prev) => !prev);
  }, []);

  const clearAIToolHighlights = useCallback(() => {
    setAIToolHighlightedNodeIds(new Set());
  }, []);

  const clearAICitationHighlights = useCallback(() => {
    setAICitationHighlightedNodeIds(new Set());
  }, []);

  const clearBlastRadius = useCallback(() => {
    setBlastRadiusNodeIds(new Set());
  }, []);

  const clearQueryHighlights = useCallback(() => {
    setHighlightedNodeIds(new Set());
    setQueryResult(null);
  }, []);

  // Node animations (for MCP tool visual feedback)
  const [animatedNodes, setAnimatedNodes] = useState<Map<string, NodeAnimation>>(new Map());
  const animationTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const triggerNodeAnimation = useCallback((nodeIds: string[], type: AnimationType) => {
    const now = Date.now();
    const duration = type === 'pulse' ? 2000 : type === 'ripple' ? 3000 : 4000;

    setAnimatedNodes((prev) => {
      const next = new Map(prev);
      for (const id of nodeIds) {
        next.set(id, { type, startTime: now, duration });
      }
      return next;
    });

    // Auto-cleanup after duration
    setTimeout(() => {
      setAnimatedNodes((prev) => {
        const next = new Map(prev);
        for (const id of nodeIds) {
          const anim = next.get(id);
          if (anim && anim.startTime === now) {
            next.delete(id);
          }
        }
        return next;
      });
    }, duration + 100);
  }, []);

  const clearAnimations = useCallback(() => {
    setAnimatedNodes(new Map());
    if (animationTimerRef.current) {
      clearInterval(animationTimerRef.current);
      animationTimerRef.current = null;
    }
  }, []);

  // Progress
  const [progress, setProgress] = useState<PipelineProgress | null>(null);

  // Project info
  const [projectName, setProjectName] = useState<string>('');

  // Multi-repo switching
  const [serverBaseUrl, setServerBaseUrl] = useState<string | null>(null);
  const [availableRepos, setAvailableRepos] = useState<BackendRepo[]>([]);
  const [repoAnalyzerRequestId, setRepoAnalyzerRequestId] = useState(0);

  // Embedding state
  const [embeddingStatus, setEmbeddingStatus] = useState<EmbeddingStatus>('idle');
  const [embeddingProgress, setEmbeddingProgress] = useState<{
    phase: string;
    percent: number;
  } | null>(null);

  const [isSettingsPanelOpen, setSettingsPanelOpen] = useState(false);

  // Code References Panel state
  const [codeReferences, setCodeReferences] = useState<CodeReference[]>([]);
  const [isCodePanelOpen, setCodePanelOpen] = useState(false);
  const [codeReferenceFocus, setCodeReferenceFocus] = useState<CodeReferenceFocus | null>(null);

  // Map of normalized file path → node ID for graph-based lookups
  const fileNodeByPath = useMemo(() => {
    if (!graph) return new Map<string, string>();
    const map = new Map<string, string>();
    for (const n of graph.nodes) {
      if (n.label === 'File') {
        map.set(normalizePath(n.properties.filePath), n.id);
      }
    }
    return map;
  }, [graph]);

  // Map of normalized path → original path for resolving partial paths
  const filePathIndex = useMemo(() => {
    if (!graph) return new Map<string, string>();
    const map = new Map<string, string>();
    for (const n of graph.nodes) {
      if (n.label === 'File' && n.properties.filePath) {
        map.set(normalizePath(n.properties.filePath), n.properties.filePath);
      }
    }
    return map;
  }, [graph]);

  const resolveFilePath = useCallback(
    (requestedPath: string): string | null => {
      const normalized = normalizePath(requestedPath);
      // Exact match
      if (filePathIndex.has(normalized)) return filePathIndex.get(normalized)!;
      // Suffix match (partial paths like "src/utils.ts")
      for (const [key, value] of filePathIndex) {
        if (key.endsWith(normalized)) return value;
      }
      return null;
    },
    [filePathIndex],
  );

  const findFileNodeId = useCallback(
    (filePath: string): string | undefined => {
      return fileNodeByPath.get(normalizePath(filePath));
    },
    [fileNodeByPath],
  );

  // Code References methods
  const addCodeReference = useCallback((ref: Omit<CodeReference, 'id'>) => {
    const id = `ref-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    const newRef: CodeReference = { ...ref, id };

    setCodeReferences((prev) => {
      // Don't add duplicates (same file + line range)
      const isDuplicate = prev.some(
        (r) =>
          r.filePath === ref.filePath && r.startLine === ref.startLine && r.endLine === ref.endLine,
      );
      if (isDuplicate) return prev;
      return [...prev, newRef];
    });

    // Auto-open panel when references are added
    setCodePanelOpen(true);

    // Signal the Code Inspector to focus (scroll + glow) this reference.
    // This should happen even if the reference already exists (duplicates are ignored),
    // so it must be separate from the add-to-list behavior.
    setCodeReferenceFocus({
      filePath: ref.filePath,
      startLine: ref.startLine,
      endLine: ref.endLine,
      ts: Date.now(),
    });

    // Track AI highlights separately so they can be toggled off in the UI
    if (ref.nodeId && ref.source === 'ai') {
      setAICitationHighlightedNodeIds((prev) => new Set([...prev, ref.nodeId!]));
    }
  }, []);

  // Remove ONLY AI-provided refs so each new chat response refreshes the Code panel
  const clearAICodeReferences = useCallback(() => {
    setCodeReferences((prev) => {
      const removed = prev.filter((r) => r.source === 'ai');
      const kept = prev.filter((r) => r.source !== 'ai');

      // Remove citation-based AI highlights for removed refs
      const removedNodeIds = new Set(removed.map((r) => r.nodeId).filter(Boolean) as string[]);
      if (removedNodeIds.size > 0) {
        setAICitationHighlightedNodeIds((prevIds) => {
          const next = new Set(prevIds);
          for (const id of removedNodeIds) next.delete(id);
          return next;
        });
      }

      // Don't auto-close if the user has something selected (top viewer)
      if (kept.length === 0 && !selectedNode) {
        setCodePanelOpen(false);
      }
      return kept;
    });
  }, [selectedNode]);

  // Auto-add a code reference when the user selects a node in the graph/tree
  useEffect(() => {
    if (!selectedNode) return;
    // User selection should show in the top "Selected file" viewer,
    // not be appended to the AI citations list.
    startTransition(() => {
      setCodePanelOpen(true);
    });
  }, [selectedNode]);

  // Backend client — direct HTTP calls (no Worker/Comlink)
  const repoRef = useRef<string | undefined>(undefined);

  const setCurrentRepo = useCallback((repoName: string) => {
    repoRef.current = repoName;
  }, []);

  const graphRef = useRef(graph);
  const fileNodeByPathRef = useRef(fileNodeByPath);
  const filePathIndexRef = useRef(filePathIndex);
  const projectNameRef = useRef(projectName);
  const embeddingStatusRef = useRef(embeddingStatus);

  useEffect(() => {
    graphRef.current = graph;
  }, [graph]);

  useEffect(() => {
    fileNodeByPathRef.current = fileNodeByPath;
  }, [fileNodeByPath]);

  useEffect(() => {
    filePathIndexRef.current = filePathIndex;
  }, [filePathIndex]);

  useEffect(() => {
    projectNameRef.current = projectName;
  }, [projectName]);

  useEffect(() => {
    embeddingStatusRef.current = embeddingStatus;
  }, [embeddingStatus]);

  const resolveFilePathForChat = useCallback((requestedPath: string): string | null => {
    const normalized = normalizePath(requestedPath);
    const index = filePathIndexRef.current;
    if (index.has(normalized)) return index.get(normalized)!;
    for (const [key, value] of index) {
      if (key.endsWith(normalized)) return value;
    }
    return null;
  }, []);

  const findFileNodeIdForChat = useCallback((filePath: string): string | undefined => {
    return fileNodeByPathRef.current.get(normalizePath(filePath));
  }, []);

  const handleFileGroundingReference = useCallback(
    (rawReference: string) => {
      const raw = rawReference.trim();
      if (!raw) return;

      let rawPath = raw;
      let startLine1: number | undefined;
      let endLine1: number | undefined;

      const lineMatch = raw.match(/^(.*):(\d+)(?:[-–](\d+))?$/);
      if (lineMatch) {
        rawPath = lineMatch[1].trim();
        startLine1 = parseInt(lineMatch[2], 10);
        endLine1 = parseInt(lineMatch[3] || lineMatch[2], 10);
      }

      const resolvedPath = resolveFilePathForChat(rawPath);
      if (!resolvedPath) return;

      const nodeId = findFileNodeIdForChat(resolvedPath);

      addCodeReference({
        filePath: resolvedPath,
        startLine: startLine1 ? Math.max(0, startLine1 - 1) : undefined,
        endLine: endLine1
          ? Math.max(0, endLine1 - 1)
          : startLine1
            ? Math.max(0, startLine1 - 1)
            : undefined,
        nodeId,
        label: 'File',
        name: resolvedPath.split('/').pop() ?? resolvedPath,
        source: 'ai',
      });
    },
    [addCodeReference, findFileNodeIdForChat, resolveFilePathForChat],
  );

  const handleNodeGroundingReference = useCallback(
    (nodeTypeAndName: string) => {
      const raw = nodeTypeAndName.trim();
      if (!raw) return;

      const graphData = graphRef.current;
      if (!graphData) return;

      const match = raw.match(
        /^(Class|Function|Method|Interface|File|Folder|Variable|Enum|Type|CodeElement):(.+)$/,
      );
      if (!match) return;

      const [, nodeType, nodeName] = match;
      const trimmedName = nodeName.trim();

      const node = graphData.nodes.find(
        (n) => n.label === nodeType && n.properties.name === trimmedName,
      );

      if (!node?.properties.filePath) return;

      const resolvedPath = resolveFilePathForChat(node.properties.filePath);
      if (!resolvedPath) return;

      addCodeReference({
        filePath: resolvedPath,
        startLine: node.properties.startLine ? node.properties.startLine - 1 : undefined,
        endLine: node.properties.endLine ? node.properties.endLine - 1 : undefined,
        nodeId: node.id,
        label: node.label,
        name: node.properties.name,
        source: 'ai',
      });
    },
    [addCodeReference, resolveFilePathForChat],
  );

  const handleTranscriptLinkClick = useCallback(
    (href: string) => {
      if (href.startsWith('code-ref:')) {
        const inner = decodeURIComponent(href.slice('code-ref:'.length));
        handleFileGroundingReference(inner);
      } else if (href.startsWith('node-ref:')) {
        const inner = decodeURIComponent(href.slice('node-ref:'.length));
        handleNodeGroundingReference(inner);
      }
    },
    [handleFileGroundingReference, handleNodeGroundingReference],
  );

  const handleChatContentGrounding = useCallback(
    (fullText: string) => {
      const fileRefRegex = new RegExp(FILE_REF_REGEX.source, FILE_REF_REGEX.flags);
      let fileMatch: RegExpExecArray | null;
      while ((fileMatch = fileRefRegex.exec(fullText)) !== null) {
        const rawPath = fileMatch[1].trim();
        const startLine = fileMatch[2];
        const endLine = fileMatch[3];
        const reference = startLine
          ? `${rawPath}:${startLine}${endLine ? `-${endLine}` : ''}`
          : rawPath;
        handleFileGroundingReference(reference);
      }

      const nodeRefRegex = new RegExp(NODE_REF_REGEX.source, NODE_REF_REGEX.flags);
      let nodeMatch: RegExpExecArray | null;
      while ((nodeMatch = nodeRefRegex.exec(fullText)) !== null) {
        handleNodeGroundingReference(`${nodeMatch[1]}:${nodeMatch[2].trim()}`);
      }
    },
    [handleFileGroundingReference, handleNodeGroundingReference],
  );

  const handleChatToolResult = useCallback((toolResult: string) => {
    const graphData = graphRef.current;
    const graphNodeIdSet = graphData ? new Set(graphData.nodes.map((n) => n.id)) : null;

    const resolveNodeIds = (rawIds: string[]) => {
      if (!graphData || !graphNodeIdSet) return new Set(rawIds);

      const matchedIds = new Set<string>();
      for (const rawId of rawIds) {
        if (graphNodeIdSet.has(rawId)) {
          matchedIds.add(rawId);
        } else {
          const found = graphData.nodes.find(
            (n) => n.id.endsWith(rawId) || n.id.endsWith(':' + rawId),
          )?.id;
          if (found) {
            matchedIds.add(found);
          }
        }
      }
      return matchedIds;
    };

    const highlightMatch = toolResult.match(/\[HIGHLIGHT_NODES:([^\]]+)\]/);
    if (highlightMatch) {
      const rawIds = highlightMatch[1]
        .split(',')
        .map((id: string) => id.trim())
        .filter(Boolean);
      if (rawIds.length > 0) {
        setAIToolHighlightedNodeIds(resolveNodeIds(rawIds));
      }
    }

    const impactMatch = toolResult.match(/\[IMPACT:([^\]]+)\]/);
    if (impactMatch) {
      const rawIds = impactMatch[1]
        .split(',')
        .map((id: string) => id.trim())
        .filter(Boolean);
      if (rawIds.length > 0) {
        setBlastRadiusNodeIds(resolveNodeIds(rawIds));
      }
    }
  }, []);

  const chatRuntimeBridge = useMemo<ChatRuntimeBridge>(
    () => ({
      getRepoName: () => repoRef.current || projectNameRef.current || undefined,
      getEmbeddingStatus: () => embeddingStatusRef.current,
      clearAICodeReferences,
      clearAIToolHighlights,
      handleContentGrounding: handleChatContentGrounding,
      handleToolResult: handleChatToolResult,
      handleTranscriptLinkClick,
    }),
    [
      clearAICodeReferences,
      clearAIToolHighlights,
      handleChatContentGrounding,
      handleChatToolResult,
      handleTranscriptLinkClick,
    ],
  );

  const requestRepoAnalyzeDialog = useCallback(() => {
    setRepoAnalyzerRequestId((prev) => prev + 1);
  }, []);

  const runQuery = useCallback(async (cypher: string): Promise<any[]> => {
    return backendRunQuery(cypher, repoRef.current);
  }, []);

  const isDatabaseReady = useCallback(async (): Promise<boolean> => {
    return probeBackend();
  }, []);

  // Embedding methods — now trigger server-side via /api/embed
  const embedAbortRef = useRef<AbortController | null>(null);

  const startEmbeddings = useCallback(async (): Promise<void> => {
    const repo = repoRef.current;
    if (!repo) throw new Error('No repository loaded');

    setEmbeddingStatus('loading');
    setEmbeddingProgress(null);

    try {
      const { jobId } = await backendStartEmbeddings(repo);

      // Stream progress via SSE
      await new Promise<void>((resolve, reject) => {
        embedAbortRef.current = streamEmbeddingProgress(
          jobId,
          (progress: JobProgress) => {
            setEmbeddingProgress({ phase: progress.phase as any, percent: progress.percent });
            if (progress.phase === 'loading-model' || progress.phase === 'loading') {
              setEmbeddingStatus('loading');
            } else if (progress.phase === 'embedding') {
              setEmbeddingStatus('embedding');
            } else if (progress.phase === 'indexing') {
              setEmbeddingStatus('indexing');
            }
          },
          () => {
            setEmbeddingStatus('ready');
            setEmbeddingProgress({ phase: 'ready' as any, percent: 100 });
            resolve();
          },
          (error: string) => {
            setEmbeddingStatus('error');
            reject(new Error(error));
          },
        );
      });
    } catch (error: any) {
      if (error?.message?.includes('already in progress')) {
        // Dedup — embeddings already running, just wait
        setEmbeddingStatus('embedding');
        return;
      }
      setEmbeddingStatus('error');
      throw error;
    }
  }, []);

  const startEmbeddingsWithFallback = useCallback(() => {
    const isPlaywright =
      (typeof navigator !== 'undefined' && navigator.webdriver) ||
      (typeof import.meta !== 'undefined' &&
        typeof import.meta.env !== 'undefined' &&
        import.meta.env.VITE_PLAYWRIGHT_TEST);
    if (isPlaywright) {
      setEmbeddingStatus('idle');
      return;
    }
    startEmbeddings().catch((err) => {
      console.warn('Embeddings auto-start failed:', err);
    });
  }, [startEmbeddings]);

  const semanticSearch = useCallback(async (query: string, k: number = 10): Promise<any[]> => {
    return backendSearch(query, { limit: k, mode: 'semantic', repo: repoRef.current });
  }, []);

  const semanticSearchWithContext = useCallback(
    async (query: string, k: number = 5, _hops: number = 2): Promise<any[]> => {
      return backendSearch(query, {
        limit: k,
        mode: 'semantic',
        enrich: true,
        repo: repoRef.current,
      });
    },
    [],
  );

  // Switch to a different repo on the connected server
  const switchRepo = useCallback(
    async (repoName: string) => {
      if (!serverBaseUrl) return;

      setProgress({
        phase: 'extracting',
        percent: 0,
        message: 'Switching repository...',
        detail: `Loading ${repoName}`,
      });
      setViewMode('loading');

      // Clear stale graph state from previous repo (highlights, selections, blast radius)
      // Without this, sigma reducers dim ALL nodes/edges because old node IDs don't match
      setHighlightedNodeIds(new Set());
      clearAIToolHighlights();
      clearAICitationHighlights();
      clearBlastRadius();
      setSelectedNode(null);
      setQueryResult(null);
      setCodeReferences([]);
      setCodePanelOpen(false);
      setCodeReferenceFocus(null);

      let connectedRepo: BackendRepo | undefined;
      let pNameStr = repoName || 'server-project';

      try {
        const result: ConnectResult = await connectToServer(
          serverBaseUrl,
          (phase, downloaded, total) => {
            if (phase === 'validating') {
              setProgress({
                phase: 'extracting',
                percent: 5,
                message: 'Switching repository...',
                detail: 'Validating',
              });
            } else if (phase === 'downloading') {
              const hasTotal = typeof total === 'number' && total > 0;
              const pct = hasTotal ? Math.round((downloaded / total) * 90) + 5 : 0;
              const mb = (downloaded / (1024 * 1024)).toFixed(1);
              setProgress({
                phase: 'extracting',
                percent: pct,
                showPercent: hasTotal,
                message: 'Downloading graph...',
                detail: `${mb} MB downloaded`,
              });
            } else if (phase === 'extracting') {
              setProgress({
                phase: 'extracting',
                percent: 97,
                message: 'Processing...',
                detail: 'Extracting file contents',
              });
            }
          },
          undefined,
          repoName,
          { awaitAnalysis: true }, // enable backend hold-queue for repos still being analyzed
        );

        // Build graph for visualization
        const repoPath = result.repoInfo.repoPath ?? result.repoInfo.path;
        // Prefer the registry name, then normalize Windows \ and Unix / paths
        const pName =
          repoName ||
          result.repoInfo.name ||
          (repoPath || '').replace(/\\/g, '/').split('/').filter(Boolean).pop() ||
          'server-project';
        setProjectName(pName);
        repoRef.current = pName;

        connectedRepo = result.repoInfo;
        pNameStr = pName;

        const newGraph = createKnowledgeGraph();
        for (const node of result.nodes) newGraph.addNode(node);
        for (const rel of result.relationships) newGraph.addRelationship(rel);
        setGraph(newGraph);
      } catch (err: unknown) {
        console.error('Repo switch failed:', err);
        setProgress({
          phase: 'error',
          percent: 0,
          message: 'Failed to switch repository',
          detail: err instanceof Error ? err.message : 'Unknown error',
        });
        setTimeout(() => {
          setViewMode('exploring');
          setProgress(null);
        }, ERROR_RESET_DELAY_MS);
        return; // Abort the whole switchRepo process
      }

      if (pNameStr) {
        // Persist the selected project in the URL so a refresh re-opens it
        const urlObj = new URL(window.location.href);
        urlObj.searchParams.set('project', pNameStr);
        window.history.replaceState(null, '', urlObj.toString());
      }

      setViewMode('exploring');
      startEmbeddingsWithFallback();
      setProgress(null);
    },
    [
      serverBaseUrl,
      setProgress,
      setViewMode,
      setProjectName,
      setGraph,
      startEmbeddingsWithFallback,
      setHighlightedNodeIds,
      clearAIToolHighlights,
      clearAICitationHighlights,
      clearBlastRadius,
      setSelectedNode,
      setQueryResult,
      setCodeReferences,
      setCodePanelOpen,
      setCodeReferenceFocus,
    ],
  );

  const removeCodeReference = useCallback(
    (id: string) => {
      setCodeReferences((prev) => {
        const ref = prev.find((r) => r.id === id);
        const newRefs = prev.filter((r) => r.id !== id);

        // Remove AI citation highlight if this was the only AI reference to that node
        if (ref?.nodeId && ref.source === 'ai') {
          const stillReferenced = newRefs.some((r) => r.nodeId === ref.nodeId && r.source === 'ai');
          if (!stillReferenced) {
            setAICitationHighlightedNodeIds((prev) => {
              const next = new Set(prev);
              next.delete(ref.nodeId!);
              return next;
            });
          }
        }

        // Auto-close panel if no references left AND no selection in top viewer
        if (newRefs.length === 0 && !selectedNode) {
          setCodePanelOpen(false);
        }

        return newRefs;
      });
    },
    [selectedNode],
  );

  const clearCodeReferences = useCallback(() => {
    setCodeReferences([]);
    setCodePanelOpen(false);
    setCodeReferenceFocus(null);
  }, []);

  const value: AppState = {
    viewMode,
    setViewMode,
    graph,
    setGraph,
    selectedNode,
    setSelectedNode,
    isRightPanelOpen,
    setRightPanelOpen,
    rightPanelTab,
    setRightPanelTab,
    openCodePanel,
    openChatPanel,
    helpDialogBoxOpen,
    setHelpDialogBoxOpen,
    visibleLabels,
    toggleLabelVisibility,
    visibleEdgeTypes,
    toggleEdgeVisibility,
    areGraphLinksVisible,
    setGraphLinksVisible,
    toggleGraphLinksVisible,
    depthFilter,
    setDepthFilter,
    highlightedNodeIds,
    setHighlightedNodeIds,
    aiCitationHighlightedNodeIds,
    aiToolHighlightedNodeIds,
    blastRadiusNodeIds,
    isAIHighlightsEnabled,
    toggleAIHighlights,
    clearAIToolHighlights,
    clearAICitationHighlights,
    clearBlastRadius,
    queryResult,
    setQueryResult,
    clearQueryHighlights,
    // Node animations
    animatedNodes,
    triggerNodeAnimation,
    clearAnimations,
    progress,
    setProgress,
    projectName,
    setProjectName,
    // Multi-repo switching
    serverBaseUrl,
    setServerBaseUrl,
    availableRepos,
    setAvailableRepos,
    switchRepo,
    setCurrentRepo,
    repoAnalyzerRequestId,
    requestRepoAnalyzeDialog,
    runQuery,
    isDatabaseReady,
    // Embedding state and methods
    embeddingStatus,
    embeddingProgress,
    startEmbeddings,
    startEmbeddingsWithFallback,
    semanticSearch,
    semanticSearchWithContext,
    isEmbeddingReady: embeddingStatus === 'ready',
    isSettingsPanelOpen,
    setSettingsPanelOpen,
    // Code References Panel
    codeReferences,
    isCodePanelOpen,
    setCodePanelOpen,
    addCodeReference,
    removeCodeReference,
    clearAICodeReferences,
    clearCodeReferences,
    codeReferenceFocus,
    chatRuntimeBridge,
  };

  return <AppStateContext.Provider value={value}>{children}</AppStateContext.Provider>;
};

export const useAppState = (): AppState => {
  const context = useContext(AppStateContext);
  if (!context) {
    throw new Error('useAppState must be used within AppStateProvider');
  }
  return context;
};
