/**
 * CSV Generator for LadybugDB Hybrid Schema
 *
 * Streams CSV rows directly to disk files in a single pass over graph nodes.
 * File contents are lazy-read from disk per-node to avoid holding the entire
 * repo in RAM. Rows are buffered (FLUSH_EVERY) before writing to minimize
 * per-row Promise overhead.
 *
 * RFC 4180 Compliant:
 * - Fields containing commas, double quotes, or newlines are enclosed in double quotes
 * - Double quotes within fields are escaped by doubling them ("")
 * - All fields are consistently quoted for safety with code content
 */

import fs from 'fs/promises';
import { createWriteStream, WriteStream } from 'fs';
import path from 'path';
import { performance } from 'node:perf_hooks';
import type { GraphNode } from 'avmatrix-shared';
import { KnowledgeGraph } from '../graph/types.js';
import { NodeTableName } from './schema.js';

/** Flush buffered rows to disk every N rows */
const FLUSH_EVERY = 500;

export interface CsvGenerationTimingBreakdown {
  contentCacheHitMs?: number;
  contentReadMs?: number;
  contentExtractMs?: number;
  rowBuildMs?: number;
  writerFlushMs?: number;
}

export interface CsvGenerationMetrics {
  timings: CsvGenerationTimingBreakdown;
  rowsByTable: Record<string, number>;
}

type CsvGenerationTimingKey = keyof CsvGenerationTimingBreakdown;

const roundCsvMs = (value: number): number => Math.round(value * 10) / 10;

const markCsvTiming = (
  timings: CsvGenerationTimingBreakdown,
  key: CsvGenerationTimingKey,
  durationMs: number,
): void => {
  if (!Number.isFinite(durationMs) || durationMs < 0) return;
  timings[key] = roundCsvMs((timings[key] ?? 0) + durationMs);
};

const timeCsvSync = <T>(
  timings: CsvGenerationTimingBreakdown,
  key: CsvGenerationTimingKey,
  fn: () => T,
): T => {
  const start = performance.now();
  try {
    return fn();
  } finally {
    markCsvTiming(timings, key, performance.now() - start);
  }
};

// ============================================================================
// CSV ESCAPE UTILITIES
// ============================================================================

export const sanitizeUTF8 = (str: string): string => {
  return str
    .replace(/\r\n/g, '\n')
    .replace(/\r/g, '\n')
    .replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, '')
    .replace(/[\uD800-\uDFFF]/g, '')
    .replace(/[\uFFFE\uFFFF]/g, '');
};

export const escapeCSVField = (value: string | number | undefined | null): string => {
  if (value === undefined || value === null) return '""';
  let str = String(value);
  str = sanitizeUTF8(str);
  return `"${str.replace(/"/g, '""')}"`;
};

export const escapeCSVNumber = (
  value: number | undefined | null,
  defaultValue: number = -1,
): string => {
  if (value === undefined || value === null) return String(defaultValue);
  return String(value);
};

// ============================================================================
// CONTENT EXTRACTION (lazy — reads from disk on demand)
// ============================================================================

export const isBinaryContent = (content: string): boolean => {
  if (!content || content.length === 0) return false;
  const sample = content.slice(0, 1000);
  let nonPrintable = 0;
  for (let i = 0; i < sample.length; i++) {
    const code = sample.charCodeAt(i);
    if (code < 9 || (code > 13 && code < 32) || code === 127) nonPrintable++;
  }
  return nonPrintable / sample.length > 0.1;
};

/**
 * LRU content cache — avoids re-reading the same source file for every
 * symbol defined in it. Sized generously so most files stay cached during
 * the single-pass node iteration.
 */
class FileContentCache {
  private cache = new Map<string, string>();
  private accessOrder: string[] = [];
  private maxSize: number;
  private repoPath: string;
  private timings: CsvGenerationTimingBreakdown;

  constructor(
    repoPath: string,
    maxSize: number = 3000,
    timings: CsvGenerationTimingBreakdown = {},
  ) {
    this.repoPath = repoPath;
    this.maxSize = maxSize;
    this.timings = timings;
  }

  async get(relativePath: string): Promise<string> {
    if (!relativePath) return '';
    const cacheStart = performance.now();
    const cached = this.cache.get(relativePath);
    if (cached !== undefined) {
      // Move to end of accessOrder (LRU promotion)
      const idx = this.accessOrder.indexOf(relativePath);
      if (idx !== -1) {
        this.accessOrder.splice(idx, 1);
        this.accessOrder.push(relativePath);
      }
      markCsvTiming(this.timings, 'contentCacheHitMs', performance.now() - cacheStart);
      return cached;
    }
    try {
      const fullPath = path.join(this.repoPath, relativePath);
      const readStart = performance.now();
      const content = await fs.readFile(fullPath, 'utf-8');
      markCsvTiming(this.timings, 'contentReadMs', performance.now() - readStart);
      this.set(relativePath, content);
      return content;
    } catch {
      markCsvTiming(this.timings, 'contentReadMs', performance.now() - cacheStart);
      this.set(relativePath, '');
      return '';
    }
  }

  private set(key: string, value: string) {
    if (this.cache.size >= this.maxSize) {
      const oldest = this.accessOrder.shift();
      if (oldest) this.cache.delete(oldest);
    }
    this.cache.set(key, value);
    this.accessOrder.push(key);
  }
}

const extractContent = async (
  node: GraphNode,
  contentCache: FileContentCache,
  timings: CsvGenerationTimingBreakdown,
): Promise<string> => {
  const filePath = node.properties.filePath;
  const content = await contentCache.get(filePath);
  return timeCsvSync(timings, 'contentExtractMs', () => {
    if (!content) return '';
    if (node.label === 'Folder') return '';
    if (isBinaryContent(content)) return '[Binary file - content not stored]';

    if (node.label === 'File') {
      const MAX_FILE_CONTENT = 10000;
      return content.length > MAX_FILE_CONTENT
        ? content.slice(0, MAX_FILE_CONTENT) + '\n... [truncated]'
        : content;
    }

    const startLine = node.properties.startLine;
    const endLine = node.properties.endLine;
    if (startLine === undefined || endLine === undefined) return '';

    const lines = content.split('\n');
    const start = Math.max(0, startLine - 2);
    const end = Math.min(lines.length - 1, endLine + 2);
    const snippet = lines.slice(start, end + 1).join('\n');
    const MAX_SNIPPET = 5000;
    return snippet.length > MAX_SNIPPET
      ? snippet.slice(0, MAX_SNIPPET) + '\n... [truncated]'
      : snippet;
  });
};

// ============================================================================
// BUFFERED CSV WRITER
// ============================================================================

class BufferedCSVWriter {
  private ws: WriteStream;
  private buffer: string[] = [];
  private timings: CsvGenerationTimingBreakdown;
  rows = 0;

  constructor(
    filePath: string,
    header: string,
    timings: CsvGenerationTimingBreakdown = {},
  ) {
    this.ws = createWriteStream(filePath, 'utf-8');
    this.timings = timings;
    // Large repos flush many times — raise listener cap to avoid MaxListenersExceededWarning
    this.ws.setMaxListeners(50);
    this.buffer.push(header);
  }

  addRow(row: string) {
    this.buffer.push(row);
    this.rows++;
    if (this.buffer.length >= FLUSH_EVERY) {
      return this.flush();
    }
    return Promise.resolve();
  }

  flush(): Promise<void> {
    if (this.buffer.length === 0) return Promise.resolve();
    const chunk = this.buffer.join('\n') + '\n';
    this.buffer.length = 0;
    const start = performance.now();
    return new Promise<void>((resolve, reject) => {
      this.ws.once('error', reject);
      const ok = this.ws.write(chunk);
      if (ok) {
        this.ws.removeListener('error', reject);
        resolve();
      } else {
        this.ws.once('drain', () => {
          this.ws.removeListener('error', reject);
          resolve();
        });
      }
    }).finally(() => {
      markCsvTiming(this.timings, 'writerFlushMs', performance.now() - start);
    });
  }

  async finish(): Promise<void> {
    await this.flush();
    return new Promise((resolve, reject) => {
      this.ws.end(() => resolve());
      this.ws.on('error', reject);
    });
  }
}

// ============================================================================
// STREAMING CSV GENERATION — SINGLE PASS
// ============================================================================

export interface StreamedCSVResult {
  nodeFiles: Map<NodeTableName, { csvPath: string; rows: number }>;
  relCsvPath: string;
  relRows: number;
  metrics: CsvGenerationMetrics;
}

const addBuiltRow = async (
  writer: BufferedCSVWriter,
  timings: CsvGenerationTimingBreakdown,
  buildRow: () => string,
): Promise<void> => {
  const row = timeCsvSync(timings, 'rowBuildMs', buildRow);
  await writer.addRow(row);
};

/**
 * Stream all CSV data directly to disk files.
 * Iterates graph nodes exactly ONCE — routes each node to the right writer.
 * File contents are lazy-read from disk with a generous LRU cache.
 */
export const streamAllCSVsToDisk = async (
  graph: KnowledgeGraph,
  repoPath: string,
  csvDir: string,
): Promise<StreamedCSVResult> => {
  // Remove stale CSVs from previous crashed runs, then recreate
  try {
    await fs.rm(csvDir, { recursive: true, force: true });
  } catch {}
  await fs.mkdir(csvDir, { recursive: true });

  // We open ~30 concurrent write-streams; raise process limit to suppress
  // MaxListenersExceededWarning (restored after all streams finish).
  const prevMax = process.getMaxListeners();
  process.setMaxListeners(prevMax + 40);

  const csvTimings: CsvGenerationTimingBreakdown = {};
  const contentCache = new FileContentCache(repoPath, 3000, csvTimings);

  // Create writers for every node type up-front
  const fileWriter = new BufferedCSVWriter(
    path.join(csvDir, 'file.csv'),
    'id,name,filePath,content',
    csvTimings,
  );
  const folderWriter = new BufferedCSVWriter(
    path.join(csvDir, 'folder.csv'),
    'id,name,filePath',
    csvTimings,
  );
  const codeElementHeader = 'id,name,filePath,startLine,endLine,isExported,content,description';
  const functionWriter = new BufferedCSVWriter(
    path.join(csvDir, 'function.csv'),
    codeElementHeader,
    csvTimings,
  );
  const classWriter = new BufferedCSVWriter(
    path.join(csvDir, 'class.csv'),
    codeElementHeader,
    csvTimings,
  );
  const interfaceWriter = new BufferedCSVWriter(
    path.join(csvDir, 'interface.csv'),
    codeElementHeader,
    csvTimings,
  );
  const methodHeader =
    'id,name,filePath,startLine,endLine,isExported,content,description,parameterCount,returnType';
  const methodWriter = new BufferedCSVWriter(
    path.join(csvDir, 'method.csv'),
    methodHeader,
    csvTimings,
  );
  const codeElemWriter = new BufferedCSVWriter(
    path.join(csvDir, 'codeelement.csv'),
    codeElementHeader,
    csvTimings,
  );
  const communityWriter = new BufferedCSVWriter(
    path.join(csvDir, 'community.csv'),
    'id,label,heuristicLabel,keywords,description,enrichedBy,cohesion,symbolCount',
    csvTimings,
  );
  const processWriter = new BufferedCSVWriter(
    path.join(csvDir, 'process.csv'),
    'id,label,heuristicLabel,processType,stepCount,communities,entryPointId,terminalId',
    csvTimings,
  );

  // Section nodes have an extra 'level' column
  const sectionWriter = new BufferedCSVWriter(
    path.join(csvDir, 'section.csv'),
    'id,name,filePath,startLine,endLine,level,content,description',
    csvTimings,
  );

  // Route nodes for API endpoint mapping
  const routeWriter = new BufferedCSVWriter(
    path.join(csvDir, 'route.csv'),
    'id,name,filePath,responseKeys,errorKeys,middleware',
    csvTimings,
  );

  // Tool nodes for MCP tool definitions
  const toolWriter = new BufferedCSVWriter(
    path.join(csvDir, 'tool.csv'),
    'id,name,filePath,description',
    csvTimings,
  );

  // Multi-language node types share the same CSV shape (no isExported column)
  const multiLangHeader = 'id,name,filePath,startLine,endLine,content,description';
  const MULTI_LANG_TYPES = [
    'Struct',
    'Enum',
    'Macro',
    'Typedef',
    'Union',
    'Namespace',
    'Trait',
    'Impl',
    'TypeAlias',
    'Const',
    'Static',
    'Variable',
    'Property',
    'Record',
    'Delegate',
    'Annotation',
    'Constructor',
    'Template',
    'Module',
  ] as const;
  const multiLangWriters = new Map<string, BufferedCSVWriter>();
  for (const t of MULTI_LANG_TYPES) {
    multiLangWriters.set(
      t,
      new BufferedCSVWriter(path.join(csvDir, `${t.toLowerCase()}.csv`), multiLangHeader, csvTimings),
    );
  }

  const codeWriterMap: Record<string, BufferedCSVWriter> = {
    Function: functionWriter,
    Class: classWriter,
    Interface: interfaceWriter,
    CodeElement: codeElemWriter,
  };

  // Deduplicate all node types — the pipeline can produce duplicate IDs across
  // all symbol types (Class, Method, Function, etc.), not just File nodes.
  // A single Set covering every label prevents PK violations on COPY.
  const seenNodeIds = new Set<string>();

  // --- SINGLE PASS over all nodes ---
  for (const node of graph.iterNodes()) {
    if (seenNodeIds.has(node.id)) continue;
    seenNodeIds.add(node.id);

    switch (node.label) {
      case 'File': {
        const content = await extractContent(node, contentCache, csvTimings);
        await addBuiltRow(fileWriter, csvTimings, () =>
          [
            escapeCSVField(node.id),
            escapeCSVField(node.properties.name || ''),
            escapeCSVField(node.properties.filePath || ''),
            escapeCSVField(content),
          ].join(','),
        );
        break;
      }
      case 'Folder':
        await addBuiltRow(folderWriter, csvTimings, () =>
          [
            escapeCSVField(node.id),
            escapeCSVField(node.properties.name || ''),
            escapeCSVField(node.properties.filePath || ''),
          ].join(','),
        );
        break;
      case 'Community': {
        const keywords = node.properties.keywords || [];
        const keywordsStr = `[${keywords.map((k: string) => `'${k.replace(/\\/g, '\\\\').replace(/'/g, "''").replace(/,/g, '\\,')}'`).join(',')}]`;
        await addBuiltRow(communityWriter, csvTimings, () =>
          [
            escapeCSVField(node.id),
            escapeCSVField(node.properties.name || ''),
            escapeCSVField(node.properties.heuristicLabel || ''),
            keywordsStr,
            escapeCSVField(node.properties.description || ''),
            escapeCSVField(node.properties.enrichedBy || 'heuristic'),
            escapeCSVNumber(node.properties.cohesion, 0),
            escapeCSVNumber(node.properties.symbolCount, 0),
          ].join(','),
        );
        break;
      }
      case 'Process': {
        const communities = node.properties.communities || [];
        const communitiesStr = `[${communities.map((c: string) => `'${c.replace(/'/g, "''")}'`).join(',')}]`;
        await addBuiltRow(processWriter, csvTimings, () =>
          [
            escapeCSVField(node.id),
            escapeCSVField(node.properties.name || ''),
            escapeCSVField(node.properties.heuristicLabel || ''),
            escapeCSVField(node.properties.processType || ''),
            escapeCSVNumber(node.properties.stepCount, 0),
            escapeCSVField(communitiesStr),
            escapeCSVField(node.properties.entryPointId || ''),
            escapeCSVField(node.properties.terminalId || ''),
          ].join(','),
        );
        break;
      }
      case 'Method': {
        const content = await extractContent(node, contentCache, csvTimings);
        await addBuiltRow(methodWriter, csvTimings, () =>
          [
            escapeCSVField(node.id),
            escapeCSVField(node.properties.name || ''),
            escapeCSVField(node.properties.filePath || ''),
            escapeCSVNumber(node.properties.startLine, -1),
            escapeCSVNumber(node.properties.endLine, -1),
            node.properties.isExported ? 'true' : 'false',
            escapeCSVField(content),
            escapeCSVField(node.properties.description || ''),
            escapeCSVNumber(node.properties.parameterCount, 0),
            escapeCSVField(node.properties.returnType || ''),
          ].join(','),
        );
        break;
      }
      case 'Section': {
        const content = await extractContent(node, contentCache, csvTimings);
        await addBuiltRow(sectionWriter, csvTimings, () =>
          [
            escapeCSVField(node.id),
            escapeCSVField(node.properties.name || ''),
            escapeCSVField(node.properties.filePath || ''),
            escapeCSVNumber(node.properties.startLine, -1),
            escapeCSVNumber(node.properties.endLine, -1),
            escapeCSVNumber(node.properties.level, 1),
            escapeCSVField(content),
            escapeCSVField(node.properties.description || ''),
          ].join(','),
        );
        break;
      }
      case 'Route': {
        const responseKeys = node.properties.responseKeys || [];
        // LadybugDB array literal inside a quoted CSV field: escapeCSVField wraps in "..."
        // and the array uses single-quoted elements
        const keysStr = `[${responseKeys.map((k: string) => `'${k.replace(/'/g, "''")}'`).join(',')}]`;
        const errorKeys = node.properties.errorKeys || [];
        const errorKeysStr = `[${errorKeys.map((k: string) => `'${k.replace(/'/g, "''")}'`).join(',')}]`;
        const middleware = node.properties.middleware || [];
        const middlewareStr = `[${middleware.map((m: string) => `'${m.replace(/'/g, "''")}'`).join(',')}]`;
        await addBuiltRow(routeWriter, csvTimings, () =>
          [
            escapeCSVField(node.id),
            escapeCSVField(node.properties.name || ''),
            escapeCSVField(node.properties.filePath || ''),
            escapeCSVField(keysStr),
            escapeCSVField(errorKeysStr),
            escapeCSVField(middlewareStr),
          ].join(','),
        );
        break;
      }
      case 'Tool':
        await addBuiltRow(toolWriter, csvTimings, () =>
          [
            escapeCSVField(node.id),
            escapeCSVField(node.properties.name || ''),
            escapeCSVField(node.properties.filePath || ''),
            escapeCSVField(node.properties.description || ''),
          ].join(','),
        );
        break;
      default: {
        // Code element nodes (Function, Class, Interface, CodeElement)
        const writer = codeWriterMap[node.label];
        if (writer) {
          const content = await extractContent(node, contentCache, csvTimings);
          await addBuiltRow(writer, csvTimings, () =>
            [
              escapeCSVField(node.id),
              escapeCSVField(node.properties.name || ''),
              escapeCSVField(node.properties.filePath || ''),
              escapeCSVNumber(node.properties.startLine, -1),
              escapeCSVNumber(node.properties.endLine, -1),
              node.properties.isExported ? 'true' : 'false',
              escapeCSVField(content),
              escapeCSVField(node.properties.description || ''),
            ].join(','),
          );
        } else {
          // Multi-language node types (Struct, Impl, Trait, Macro, etc.)
          const mlWriter = multiLangWriters.get(node.label);
          if (mlWriter) {
            const content = await extractContent(node, contentCache, csvTimings);
            await addBuiltRow(mlWriter, csvTimings, () =>
              [
                escapeCSVField(node.id),
                escapeCSVField(node.properties.name || ''),
                escapeCSVField(node.properties.filePath || ''),
                escapeCSVNumber(node.properties.startLine, -1),
                escapeCSVNumber(node.properties.endLine, -1),
                escapeCSVField(content),
                escapeCSVField(node.properties.description || ''),
              ].join(','),
            );
          }
        }
        break;
      }
    }
  }

  // Finish all node writers
  const allWriters = [
    fileWriter,
    folderWriter,
    functionWriter,
    classWriter,
    interfaceWriter,
    methodWriter,
    codeElemWriter,
    communityWriter,
    processWriter,
    sectionWriter,
    routeWriter,
    toolWriter,
    ...multiLangWriters.values(),
  ];
  await Promise.all(allWriters.map((w) => w.finish()));

  // --- Stream relationship CSV ---
  const relCsvPath = path.join(csvDir, 'relations.csv');
  const relWriter = new BufferedCSVWriter(
    relCsvPath,
    'from,to,type,confidence,reason,step',
    csvTimings,
  );
  for (const rel of graph.iterRelationships()) {
    await addBuiltRow(relWriter, csvTimings, () =>
      [
        escapeCSVField(rel.sourceId),
        escapeCSVField(rel.targetId),
        escapeCSVField(rel.type),
        escapeCSVNumber(rel.confidence, 1.0),
        escapeCSVField(rel.reason),
        escapeCSVNumber((rel as any).step, 0),
      ].join(','),
    );
  }
  await relWriter.finish();

  // Build result map — only include tables that have rows
  const nodeFiles = new Map<NodeTableName, { csvPath: string; rows: number }>();
  const rowsByTable: Record<string, number> = {};
  const tableMap: [NodeTableName, BufferedCSVWriter][] = [
    ['File', fileWriter],
    ['Folder', folderWriter],
    ['Function', functionWriter],
    ['Class', classWriter],
    ['Interface', interfaceWriter],
    ['Method', methodWriter],
    ['CodeElement', codeElemWriter],
    ['Community', communityWriter],
    ['Process', processWriter],
    ['Section' as NodeTableName, sectionWriter],
    ['Route' as NodeTableName, routeWriter],
    ['Tool' as NodeTableName, toolWriter],
    ...Array.from(multiLangWriters.entries()).map(
      ([name, w]) => [name as NodeTableName, w] as [NodeTableName, BufferedCSVWriter],
    ),
  ];
  for (const [name, writer] of tableMap) {
    if (writer.rows > 0) {
      rowsByTable[name] = writer.rows;
      nodeFiles.set(name, {
        csvPath: path.join(csvDir, `${name.toLowerCase()}.csv`),
        rows: writer.rows,
      });
    }
  }
  rowsByTable.Relationship = relWriter.rows;

  // Restore original process listener limit
  process.setMaxListeners(prevMax);

  return {
    nodeFiles,
    relCsvPath,
    relRows: relWriter.rows,
    metrics: { timings: csvTimings, rowsByTable },
  };
};
