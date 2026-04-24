/**
 * Community Detection Processor
 *
 * Uses the Leiden algorithm (via graphology-communities-leiden) to detect
 * communities/clusters in the code graph based on CALLS relationships.
 *
 * Communities represent groups of code that work together frequently,
 * helping agents navigate the codebase by functional area rather than file structure.
 */

// NOTE: The Leiden algorithm source is vendored from graphology's repo
// (src/communities-leiden) because it was never published to npm.
// We use createRequire to load the CommonJS vendored files in ESM context.
import Graph from 'graphology';
import type { AbstractGraph, Attributes } from 'graphology-types';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import type { GraphNode, GraphRelationship, NodeLabel } from 'avmatrix-shared';
import { KnowledgeGraph } from '../graph/types.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
// Navigate to package root (works from both src/ and dist/)
const leidenPath = resolve(__dirname, '..', '..', '..', 'vendor', 'leiden', 'index.cjs');
const _require = createRequire(import.meta.url);
/** Graphology Graph instance type (AbstractGraph from graphology-types avoids CJS/ESM interop namespace issue) */
type GraphInstance = AbstractGraph<Attributes, Attributes, Attributes>;

const leiden: LeidenModule = _require(leidenPath);
const LEIDEN_RNG_SEED = 0x5eed_c0de;

/** Vendored Leiden algorithm module shape */
interface LeidenModule {
  detailed: (graph: GraphInstance, options: Record<string, unknown>) => LeidenDetailedResult;
}

/** Result returned by leiden.detailed() */
interface LeidenDetailedResult {
  communities: Record<string, number>;
  count: number;
  modularity: number;
}

// ============================================================================
// TYPES
// ============================================================================

export interface CommunityNode {
  id: string;
  label: string;
  heuristicLabel: string;
  cohesion: number;
  symbolCount: number;
}

export interface CommunityMembership {
  nodeId: string;
  communityId: string;
}

export interface CommunityDetectionResult {
  communities: CommunityNode[];
  memberships: CommunityMembership[];
  stats: {
    totalCommunities: number;
    modularity: number;
    nodesProcessed: number;
  };
}

// ============================================================================
// COMMUNITY COLORS (for visualization)
// ============================================================================

export const COMMUNITY_COLORS = [
  '#ef4444', // red
  '#f97316', // orange
  '#eab308', // yellow
  '#22c55e', // green
  '#06b6d4', // cyan
  '#3b82f6', // blue
  '#8b5cf6', // violet
  '#d946ef', // fuchsia
  '#ec4899', // pink
  '#f43f5e', // rose
  '#14b8a6', // teal
  '#84cc16', // lime
];

export const getCommunityColor = (communityIndex: number): string => {
  return COMMUNITY_COLORS[communityIndex % COMMUNITY_COLORS.length];
};

// ============================================================================
// MAIN PROCESSOR
// ============================================================================

/**
 * Detect communities in the knowledge graph using Leiden algorithm
 *
 * This runs AFTER all relationships (CALLS, IMPORTS, etc.) have been built.
 * It uses primarily CALLS edges to cluster code that works together.
 */
export const processCommunities = async (
  knowledgeGraph: KnowledgeGraph,
  onProgress?: (message: string, progress: number) => void,
): Promise<CommunityDetectionResult> => {
  onProgress?.('Building graph for community detection...', 0);

  // Pre-check total symbol count to determine large-graph mode before building
  let symbolCount = 0;
  knowledgeGraph.forEachNode((node) => {
    if (
      node.label === 'Function' ||
      node.label === 'Class' ||
      node.label === 'Method' ||
      node.label === 'Interface'
    ) {
      symbolCount++;
    }
  });
  const isLarge = symbolCount > 10_000;

  const graph = buildGraphologyGraph(knowledgeGraph, isLarge);

  if (graph.order === 0) {
    return {
      communities: [],
      memberships: [],
      stats: { totalCommunities: 0, modularity: 0, nodesProcessed: 0 },
    };
  }

  const nodeCount = graph.order;
  const edgeCount = graph.size;

  onProgress?.(
    `Running Leiden on ${nodeCount} nodes, ${edgeCount} edges${isLarge ? ` (filtered from ${symbolCount} symbols)` : ''}...`,
    30,
  );

  // Large graphs: higher resolution + capped iterations (matching Python leidenalg default of 2).
  // The first 2 iterations capture most modularity; additional iterations have diminishing returns.
  // Full analyze must not return degraded community output, so this does not time out/fallback.
  const details = leiden.detailed(graph, {
    resolution: isLarge ? 2.0 : 1.0,
    maxIterations: isLarge ? 3 : 0,
    rng: createSeededRng(LEIDEN_RNG_SEED),
  });

  onProgress?.(`Found ${details.count} communities...`, 60);

  // Step 3: Create community nodes with heuristic labels
  const communityNodes = createCommunityNodes(
    details.communities as Record<string, number>,
    details.count,
    graph,
    knowledgeGraph,
  );

  onProgress?.('Creating membership edges...', 80);

  // Step 4: Create membership mappings
  const memberships: CommunityMembership[] = [];
  sortedCommunityEntries(details.communities).forEach(([nodeId, communityNum]) => {
    memberships.push({
      nodeId,
      communityId: `comm_${communityNum}`,
    });
  });

  onProgress?.('Community detection complete!', 100);

  return {
    communities: communityNodes,
    memberships,
    stats: {
      totalCommunities: details.count,
      modularity: details.modularity,
      nodesProcessed: graph.order,
    },
  };
};

// ============================================================================
// HELPER: Build graphology graph from knowledge graph
// ============================================================================

/**
 * Build a graphology graph containing only symbol nodes and clustering edges.
 * For large graphs (>10K symbols), filter out low-confidence fuzzy-global edges
 * and degree-1 nodes that add noise and massively increase Leiden runtime.
 */
const MIN_CONFIDENCE_LARGE = 0.5;

const buildGraphologyGraph = (knowledgeGraph: KnowledgeGraph, isLarge: boolean): GraphInstance => {
  const GraphCtor = Graph as unknown as new (options: {
    type: string;
    allowSelfLoops: boolean;
  }) => GraphInstance;
  const graph = new GraphCtor({ type: 'undirected', allowSelfLoops: false });

  const symbolTypes = new Set<NodeLabel>(['Function', 'Class', 'Method', 'Interface']);
  const clusteringRelTypes = new Set(['CALLS', 'EXTENDS', 'IMPLEMENTS']);
  const connectedNodes = new Set<string>();
  const nodeDegree = new Map<string, number>();

  const clusteringRelationships = Array.from(knowledgeGraph.iterRelationships())
    .filter((rel) => {
      if (!clusteringRelTypes.has(rel.type) || rel.sourceId === rel.targetId) return false;
      if (isLarge && rel.confidence < MIN_CONFIDENCE_LARGE) return false;
      return true;
    })
    .sort(compareRelationshipsForCommunityGraph);

  clusteringRelationships.forEach((rel) => {
    if (!clusteringRelTypes.has(rel.type) || rel.sourceId === rel.targetId) return;
    if (isLarge && rel.confidence < MIN_CONFIDENCE_LARGE) return;

    connectedNodes.add(rel.sourceId);
    connectedNodes.add(rel.targetId);
    nodeDegree.set(rel.sourceId, (nodeDegree.get(rel.sourceId) || 0) + 1);
    nodeDegree.set(rel.targetId, (nodeDegree.get(rel.targetId) || 0) + 1);
  });

  Array.from(knowledgeGraph.iterNodes())
    .filter((node) => {
      if (!symbolTypes.has(node.label) || !connectedNodes.has(node.id)) return false;
      // For large graphs, skip degree-1 nodes — they just become singletons or
      // get absorbed into their single neighbor's community, but cost iteration time.
      if (isLarge && (nodeDegree.get(node.id) || 0) < 2) return false;
      return true;
    })
    .sort(compareNodesForCommunityGraph)
    .forEach((node) => {
      graph.addNode(node.id, {
        name: node.properties.name,
        filePath: node.properties.filePath,
        type: node.label,
      });
    });

  clusteringRelationships.forEach((rel) => {
    if (!clusteringRelTypes.has(rel.type)) return;
    if (isLarge && rel.confidence < MIN_CONFIDENCE_LARGE) return;
    if (
      graph.hasNode(rel.sourceId) &&
      graph.hasNode(rel.targetId) &&
      rel.sourceId !== rel.targetId
    ) {
      if (!graph.hasEdge(rel.sourceId, rel.targetId)) {
        graph.addEdge(rel.sourceId, rel.targetId);
      }
    }
  });

  return graph;
};

// ============================================================================
// HELPER: Create community nodes with heuristic labels
// ============================================================================

/**
 * Create Community nodes with auto-generated labels based on member file paths
 */
const createCommunityNodes = (
  communities: Record<string, number>,
  communityCount: number,
  graph: GraphInstance,
  knowledgeGraph: KnowledgeGraph,
): CommunityNode[] => {
  // Group node IDs by community
  const communityMembers = new Map<number, string[]>();

  sortedCommunityEntries(communities).forEach(([nodeId, commNum]) => {
    if (!communityMembers.has(commNum)) {
      communityMembers.set(commNum, []);
    }
    communityMembers.get(commNum)!.push(nodeId);
  });

  // Build node lookup for file paths
  const nodePathMap = new Map<string, string>();
  for (const node of knowledgeGraph.iterNodes()) {
    if (node.properties.filePath) {
      nodePathMap.set(node.id, node.properties.filePath);
    }
  }

  // Create community nodes - SKIP SINGLETONS (isolated nodes)
  const communityNodes: CommunityNode[] = [];

  Array.from(communityMembers.entries())
    .sort(([a], [b]) => a - b)
    .forEach(([commNum, memberIds]) => {
      memberIds.sort(compareIds);
      // Skip singleton communities - they're just isolated nodes
      if (memberIds.length < 2) return;

      const heuristicLabel = generateHeuristicLabel(memberIds, nodePathMap, graph, commNum);

      communityNodes.push({
        id: `comm_${commNum}`,
        label: heuristicLabel,
        heuristicLabel,
        cohesion: calculateCohesion(memberIds, graph),
        symbolCount: memberIds.length,
      });
    });

  communityNodes.sort(compareCommunityNodes);

  return communityNodes;
};

// ============================================================================
// HELPER: Generate heuristic label from folder patterns
// ============================================================================

/**
 * Generate a human-readable label from the most common folder name in the community
 */
const generateHeuristicLabel = (
  memberIds: string[],
  nodePathMap: Map<string, string>,
  graph: GraphInstance,
  commNum: number,
): string => {
  // Collect folder names from file paths
  const folderCounts = new Map<string, number>();

  memberIds.forEach((nodeId) => {
    const filePath = nodePathMap.get(nodeId) || '';
    const parts = filePath.split('/').filter(Boolean);

    // Get the most specific folder (parent directory)
    if (parts.length >= 2) {
      const folder = parts[parts.length - 2];
      // Skip generic folder names
      if (
        !['src', 'lib', 'core', 'utils', 'common', 'shared', 'helpers'].includes(
          folder.toLowerCase(),
        )
      ) {
        folderCounts.set(folder, (folderCounts.get(folder) || 0) + 1);
      }
    }
  });

  // Find most common folder
  let maxCount = 0;
  let bestFolder = '';

  Array.from(folderCounts.entries())
    .sort(([folderA], [folderB]) => folderA.localeCompare(folderB))
    .forEach(([folder, count]) => {
      if (count > maxCount) {
        maxCount = count;
        bestFolder = folder;
      }
    });

  if (bestFolder) {
    // Capitalize first letter
    return bestFolder.charAt(0).toUpperCase() + bestFolder.slice(1);
  }

  // Fallback: use function names to detect patterns
  const names: string[] = [];
  memberIds.forEach((nodeId) => {
    const name = graph.getNodeAttribute(nodeId, 'name');
    if (name) names.push(name);
  });

  // Look for common prefixes
  if (names.length > 2) {
    const commonPrefix = findCommonPrefix(names);
    if (commonPrefix.length > 2) {
      return commonPrefix.charAt(0).toUpperCase() + commonPrefix.slice(1);
    }
  }

  // Last resort: generic name with community ID for uniqueness
  return `Cluster_${commNum}`;
};

/**
 * Find common prefix among strings
 */
const findCommonPrefix = (strings: string[]): string => {
  if (strings.length === 0) return '';

  const sorted = strings.slice().sort();
  const first = sorted[0];
  const last = sorted[sorted.length - 1];

  let i = 0;
  while (i < first.length && first[i] === last[i]) {
    i++;
  }

  return first.substring(0, i);
};

// ============================================================================
// HELPER: Calculate community cohesion
// ============================================================================

/**
 * Estimate cohesion score (0-1) based on internal edge density.
 * Uses sampling for large communities to avoid O(N^2) cost.
 */
const calculateCohesion = (memberIds: string[], graph: GraphInstance): number => {
  if (memberIds.length <= 1) return 1.0;

  const memberSet = new Set(memberIds);

  // Sample up to 50 members for large communities
  const SAMPLE_SIZE = 50;
  const sample = memberIds.length <= SAMPLE_SIZE ? memberIds : memberIds.slice(0, SAMPLE_SIZE);

  let internalEdges = 0;
  let totalEdges = 0;

  for (const nodeId of sample) {
    if (!graph.hasNode(nodeId)) continue;
    graph.forEachNeighbor(nodeId, (neighbor: string) => {
      totalEdges++;
      if (memberSet.has(neighbor)) {
        internalEdges++;
      }
    });
  }

  // Cohesion = fraction of edges that stay internal
  if (totalEdges === 0) return 1.0;
  return Math.min(1.0, internalEdges / totalEdges);
};

const sortedCommunityEntries = (communities: Record<string, number>): [string, number][] =>
  Object.entries(communities).sort(([nodeA, commA], [nodeB, commB]) => {
    const commDiff = commA - commB;
    if (commDiff !== 0) return commDiff;
    return nodeA.localeCompare(nodeB);
  });

const compareIds = (a: string, b: string): number => a.localeCompare(b);

const compareNodesForCommunityGraph = (a: GraphNode, b: GraphNode): number =>
  a.id.localeCompare(b.id);

const compareRelationshipsForCommunityGraph = (
  a: GraphRelationship,
  b: GraphRelationship,
): number =>
  a.type.localeCompare(b.type) ||
  a.sourceId.localeCompare(b.sourceId) ||
  a.targetId.localeCompare(b.targetId) ||
  a.id.localeCompare(b.id);

const compareCommunityNodes = (a: CommunityNode, b: CommunityNode): number =>
  b.symbolCount - a.symbolCount ||
  a.heuristicLabel.localeCompare(b.heuristicLabel) ||
  a.id.localeCompare(b.id);

const createSeededRng = (seed: number): (() => number) => {
  let state = seed >>> 0;
  return () => {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
};
