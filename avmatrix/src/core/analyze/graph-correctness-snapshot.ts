import crypto from 'node:crypto';
import type { GraphNode, GraphRelationship } from 'avmatrix-shared';
import type { KnowledgeGraph } from '../graph/types.js';
import type { PipelineResult } from '../../types/pipeline.js';

export interface GraphCorrectnessSnapshot {
  totalFileCount?: number;
  nodeCount: number;
  relationshipCount: number;
  communityCount?: number;
  processCount?: number;
  byNodeLabel: Record<string, number>;
  byRelationshipType: Record<string, number>;
  nodeDigest: string;
  relationshipDigest: string;
  usedWorkerPool?: boolean;
  crossFileReprocessedFiles?: number;
}

export interface GraphCorrectnessDiff {
  field: string;
  expected: unknown;
  actual: unknown;
}

export function createGraphCorrectnessSnapshot(
  input: KnowledgeGraph | PipelineResult,
): GraphCorrectnessSnapshot {
  const graph = isPipelineResult(input) ? input.graph : input;
  const byNodeLabel: Record<string, number> = {};
  const byRelationshipType: Record<string, number> = {};
  const nodeLines: string[] = [];
  const relationshipLines: string[] = [];

  graph.forEachNode((node) => {
    byNodeLabel[node.label] = (byNodeLabel[node.label] ?? 0) + 1;
    nodeLines.push(nodeSnapshotLine(node));
  });

  graph.forEachRelationship((relationship) => {
    byRelationshipType[relationship.type] = (byRelationshipType[relationship.type] ?? 0) + 1;
    relationshipLines.push(relationshipSnapshotLine(relationship));
  });

  nodeLines.sort();
  relationshipLines.sort();

  const snapshot: GraphCorrectnessSnapshot = {
    nodeCount: graph.nodeCount,
    relationshipCount: graph.relationshipCount,
    byNodeLabel: sortNumericRecord(byNodeLabel),
    byRelationshipType: sortNumericRecord(byRelationshipType),
    nodeDigest: hashLines(nodeLines),
    relationshipDigest: hashLines(relationshipLines),
  };

  if (isPipelineResult(input)) {
    snapshot.totalFileCount = input.totalFileCount;
    snapshot.communityCount = input.communityResult?.stats.totalCommunities;
    snapshot.processCount = input.processResult?.stats.totalProcesses;
    snapshot.usedWorkerPool = input.usedWorkerPool;
    snapshot.crossFileReprocessedFiles = input.performance?.counters.crossFileReprocessedFiles;
  }

  return snapshot;
}

export function compareGraphCorrectnessSnapshots(
  expected: GraphCorrectnessSnapshot,
  actual: GraphCorrectnessSnapshot,
): GraphCorrectnessDiff[] {
  const diffs: GraphCorrectnessDiff[] = [];
  compareField(diffs, 'totalFileCount', expected.totalFileCount, actual.totalFileCount);
  compareField(diffs, 'nodeCount', expected.nodeCount, actual.nodeCount);
  compareField(diffs, 'relationshipCount', expected.relationshipCount, actual.relationshipCount);
  compareField(diffs, 'communityCount', expected.communityCount, actual.communityCount);
  compareField(diffs, 'processCount', expected.processCount, actual.processCount);
  compareField(diffs, 'byNodeLabel', expected.byNodeLabel, actual.byNodeLabel);
  compareField(diffs, 'byRelationshipType', expected.byRelationshipType, actual.byRelationshipType);
  compareField(diffs, 'nodeDigest', expected.nodeDigest, actual.nodeDigest);
  compareField(diffs, 'relationshipDigest', expected.relationshipDigest, actual.relationshipDigest);
  compareField(diffs, 'usedWorkerPool', expected.usedWorkerPool, actual.usedWorkerPool);
  compareField(
    diffs,
    'crossFileReprocessedFiles',
    expected.crossFileReprocessedFiles,
    actual.crossFileReprocessedFiles,
  );
  return diffs;
}

function isPipelineResult(input: KnowledgeGraph | PipelineResult): input is PipelineResult {
  return 'graph' in input && typeof input.totalFileCount === 'number';
}

function nodeSnapshotLine(node: GraphNode): string {
  return stableStringify({
    id: node.id,
    label: node.label,
    properties: stableValue(node.properties),
  });
}

function relationshipSnapshotLine(relationship: GraphRelationship): string {
  return stableStringify({
    id: relationship.id,
    type: relationship.type,
    sourceId: relationship.sourceId,
    targetId: relationship.targetId,
    confidence: relationship.confidence,
    reason: relationship.reason,
    step: relationship.step,
  });
}

function hashLines(lines: string[]): string {
  return crypto.createHash('sha256').update(lines.join('\n')).digest('hex');
}

function compareField(
  diffs: GraphCorrectnessDiff[],
  field: string,
  expected: unknown,
  actual: unknown,
): void {
  if (stableStringify(expected) !== stableStringify(actual)) {
    diffs.push({ field, expected, actual });
  }
}

function sortNumericRecord(record: Record<string, number>): Record<string, number> {
  const out: Record<string, number> = {};
  for (const key of Object.keys(record).sort()) out[key] = record[key];
  return out;
}

function stableValue(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(stableValue);
  if (value && typeof value === 'object') {
    const out: Record<string, unknown> = {};
    for (const key of Object.keys(value as Record<string, unknown>).sort()) {
      out[key] = stableValue((value as Record<string, unknown>)[key]);
    }
    return out;
  }
  return value;
}

function stableStringify(value: unknown): string {
  return JSON.stringify(stableValue(value));
}
