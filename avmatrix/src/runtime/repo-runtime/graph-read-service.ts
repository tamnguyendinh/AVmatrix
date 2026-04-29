import { NODE_TABLES, type GraphNode, type GraphRelationship } from 'avmatrix-shared';
import {
  executeRepoReadQuery,
  streamRepoReadQuery,
  type RepoReadTarget as RepoGraphTarget,
} from './repo-read-executor.js';

export type GraphStreamRecord =
  | { type: 'node'; data: GraphNode }
  | { type: 'relationship'; data: GraphRelationship }
  | { type: 'error'; error: string };

export interface GraphStreamSink {
  write(record: GraphStreamRecord): Promise<void>;
  flush?(): Promise<void>;
}

export type GraphQueryExecutor = (cypher: string) => Promise<any[]>;
export type GraphQueryStreamer = (
  cypher: string,
  onRow: (row: any) => void | Promise<void>,
) => Promise<number>;

export const isIgnorableGraphQueryError = (err: unknown): boolean => {
  const message = err instanceof Error ? err.message : String(err);
  return (
    message.includes('does not exist') ||
    message.includes('not found') ||
    message.includes('No table named')
  );
};

export const GRAPH_RELATIONSHIP_QUERY =
  `MATCH (a)-[r:CodeRelation]->(b) RETURN a.id AS sourceId, b.id AS targetId, ` +
  `r.type AS type, r.confidence AS confidence, r.reason AS reason, r.step AS step`;

export const quoteNodeTable = (table: string): string => `\`${table.replace(/`/g, '``')}\``;

export const getNodeQuery = (table: string, includeContent: boolean): string => {
  const tableLabel = quoteNodeTable(table);

  if (table === 'File') {
    return includeContent
      ? `MATCH (n:${tableLabel}) RETURN n.id AS id, n.name AS name, n.filePath AS filePath, n.content AS content`
      : `MATCH (n:${tableLabel}) RETURN n.id AS id, n.name AS name, n.filePath AS filePath`;
  }
  if (table === 'Folder') {
    return `MATCH (n:${tableLabel}) RETURN n.id AS id, n.name AS name, n.filePath AS filePath`;
  }
  if (table === 'Community') {
    return `MATCH (n:${tableLabel}) RETURN n.id AS id, n.label AS label, n.heuristicLabel AS heuristicLabel, n.cohesion AS cohesion, n.symbolCount AS symbolCount`;
  }
  if (table === 'Process') {
    return `MATCH (n:${tableLabel}) RETURN n.id AS id, n.label AS label, n.heuristicLabel AS heuristicLabel, n.processType AS processType, n.stepCount AS stepCount, n.communities AS communities, n.entryPointId AS entryPointId, n.terminalId AS terminalId`;
  }
  if (table === 'Route') {
    return `MATCH (n:${tableLabel}) RETURN n.id AS id, n.name AS name, n.filePath AS filePath, n.responseKeys AS responseKeys, n.errorKeys AS errorKeys, n.middleware AS middleware`;
  }
  if (table === 'Tool') {
    return `MATCH (n:${tableLabel}) RETURN n.id AS id, n.name AS name, n.filePath AS filePath, n.description AS description`;
  }
  return includeContent
    ? `MATCH (n:${tableLabel}) RETURN n.id AS id, n.name AS name, n.filePath AS filePath, n.startLine AS startLine, n.endLine AS endLine, n.content AS content`
    : `MATCH (n:${tableLabel}) RETURN n.id AS id, n.name AS name, n.filePath AS filePath, n.startLine AS startLine, n.endLine AS endLine`;
};

export const mapGraphNodeRow = (table: string, row: any, includeContent: boolean): GraphNode => ({
  id: row.id ?? row[0],
  label: table as GraphNode['label'],
  properties: {
    name: row.name ?? row.label ?? row[1],
    filePath: row.filePath ?? row[2],
    startLine: row.startLine,
    endLine: row.endLine,
    content: includeContent ? row.content : undefined,
    responseKeys: row.responseKeys,
    errorKeys: row.errorKeys,
    middleware: row.middleware,
    heuristicLabel: row.heuristicLabel,
    cohesion: row.cohesion,
    symbolCount: row.symbolCount,
    description: row.description,
    processType: row.processType,
    stepCount: row.stepCount,
    communities: row.communities,
    entryPointId: row.entryPointId,
    terminalId: row.terminalId,
  } as GraphNode['properties'],
});

export const mapGraphRelationshipRow = (row: any): GraphRelationship => ({
  id: `${row.sourceId}_${row.type}_${row.targetId}`,
  type: row.type,
  sourceId: row.sourceId,
  targetId: row.targetId,
  confidence: row.confidence,
  reason: row.reason,
  step: row.step,
});

export const buildGraphFromExecutor = async (
  executeQuery: GraphQueryExecutor,
  includeContent = false,
): Promise<{ nodes: GraphNode[]; relationships: GraphRelationship[] }> => {
  const nodes: GraphNode[] = [];
  for (const table of NODE_TABLES) {
    try {
      const rows = await executeQuery(getNodeQuery(table, includeContent));
      for (const row of rows) {
        nodes.push(mapGraphNodeRow(table, row, includeContent));
      }
    } catch (err) {
      if (!isIgnorableGraphQueryError(err)) {
        throw err;
      }
    }
  }

  const relationships: GraphRelationship[] = [];
  const relRows = await executeQuery(GRAPH_RELATIONSHIP_QUERY);
  for (const row of relRows) {
    relationships.push(mapGraphRelationshipRow(row));
  }

  return { nodes, relationships };
};

export const streamGraphFromExecutor = async (
  streamQuery: GraphQueryStreamer,
  sink: GraphStreamSink,
  includeContent = false,
): Promise<void> => {
  for (const table of NODE_TABLES) {
    try {
      await streamQuery(getNodeQuery(table, includeContent), async (row) => {
        await sink.write({
          type: 'node',
          data: mapGraphNodeRow(table, row, includeContent),
        });
      });
      await sink.flush?.();
    } catch (err) {
      if (!isIgnorableGraphQueryError(err)) {
        throw err;
      }
    }
  }

  await streamQuery(GRAPH_RELATIONSHIP_QUERY, async (row) => {
    await sink.write({
      type: 'relationship',
      data: mapGraphRelationshipRow(row),
    });
  });
  await sink.flush?.();
};

export const buildRepoGraph = async (
  target: RepoGraphTarget,
  includeContent = false,
): Promise<{ nodes: GraphNode[]; relationships: GraphRelationship[] }> => {
  return buildGraphFromExecutor((cypher) => executeRepoReadQuery(target, cypher), includeContent);
};

export const streamRepoGraph = async (
  target: RepoGraphTarget,
  sink: GraphStreamSink,
  includeContent = false,
): Promise<void> => {
  await streamGraphFromExecutor(
    (cypher, onRow) => streamRepoReadQuery(target, cypher, onRow),
    sink,
    includeContent,
  );
};
