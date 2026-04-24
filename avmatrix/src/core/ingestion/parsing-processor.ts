import { getLanguageFromFilename, type NodeLabel, type ParsedFile } from 'avmatrix-shared';
import type { KnowledgeGraph } from '../graph/types.js';
import type { SymbolTableWriter, ExtractedHeritage } from './model/index.js';
import type { WorkerPool } from './workers/worker-pool.js';
import type {
  ParseWorkerResult,
  ParseWorkerInput,
  ExtractedImport,
  ExtractedCall,
  ExtractedAssignment,
  ExtractedRoute,
  ExtractedFetchCall,
  ExtractedDecoratorRoute,
  ExtractedToolDef,
  FileConstructorBindings,
  FileScopeBindings,
  ExtractedORMQuery,
} from './workers/parse-worker.js';

export type FileProgressCallback = (current: number, total: number, filePath: string) => void;

export interface WorkerExtractedData {
  imports: ExtractedImport[];
  calls: ExtractedCall[];
  assignments: ExtractedAssignment[];
  heritage: ExtractedHeritage[];
  routes: ExtractedRoute[];
  fetchCalls: ExtractedFetchCall[];
  decoratorRoutes: ExtractedDecoratorRoute[];
  toolDefs: ExtractedToolDef[];
  ormQueries: ExtractedORMQuery[];
  constructorBindings: FileConstructorBindings[];
  fileScopeBindings: FileScopeBindings[];
  /**
   * Per-file `ParsedFile` artifacts from the new scope-based resolution
   * pipeline (RFC #909 Ring 2). Empty until a provider implements
   * `emitScopeCaptures` — additive to the legacy DAG path.
   */
  parsedFiles: ParsedFile[];
}

const emptyWorkerExtractedData = (): WorkerExtractedData => ({
  imports: [],
  calls: [],
  assignments: [],
  heritage: [],
  routes: [],
  fetchCalls: [],
  decoratorRoutes: [],
  toolDefs: [],
  ormQueries: [],
  constructorBindings: [],
  fileScopeBindings: [],
  parsedFiles: [],
});

/**
 * Worker-only parser entry point.
 *
 * Full analyze now treats parse workers as the canonical path. This function
 * deliberately does not catch worker failures or fall back to the removed
 * sequential parser: callers must retry/isolate in the worker scheduler or fail
 * analyze with a clear diagnostic.
 */
export const processParsing = async (
  graph: KnowledgeGraph,
  files: { path: string; content: string }[],
  symbolTable: SymbolTableWriter,
  workerPool: WorkerPool,
  onFileProgress?: FileProgressCallback,
): Promise<WorkerExtractedData> => {
  if (!workerPool) {
    throw new Error(
      'Parse worker pool is required. Sequential parsing has been removed from full analyze.',
    );
  }

  const parseableFiles: ParseWorkerInput[] = [];
  for (const file of files) {
    const lang = getLanguageFromFilename(file.path);
    if (lang) parseableFiles.push({ path: file.path, content: file.content });
  }

  if (parseableFiles.length === 0) return emptyWorkerExtractedData();

  const total = files.length;
  const chunkResults = await workerPool.dispatch<ParseWorkerInput, ParseWorkerResult>(
    parseableFiles,
    (filesProcessed) => {
      onFileProgress?.(Math.min(filesProcessed, total), total, 'Parsing...');
    },
    {
      getItemPath: (file) => file.path,
      getItemSize: (file) => Buffer.byteLength(file.content),
      verbose: process.env.AVMATRIX_VERBOSE === '1',
    },
  );

  const allImports: ExtractedImport[] = [];
  const allCalls: ExtractedCall[] = [];
  const allAssignments: ExtractedAssignment[] = [];
  const allHeritage: ExtractedHeritage[] = [];
  const allRoutes: ExtractedRoute[] = [];
  const allFetchCalls: ExtractedFetchCall[] = [];
  const allDecoratorRoutes: ExtractedDecoratorRoute[] = [];
  const allToolDefs: ExtractedToolDef[] = [];
  const allORMQueries: ExtractedORMQuery[] = [];
  const allConstructorBindings: FileConstructorBindings[] = [];
  const fileScopeBindingsByFile: FileScopeBindings[] = [];
  const allParsedFiles: ParsedFile[] = [];

  for (const result of chunkResults) {
    for (const node of result.nodes) {
      graph.addNode({
        id: node.id,
        label: node.label as NodeLabel,
        properties: node.properties,
      });
    }

    for (const rel of result.relationships) {
      graph.addRelationship(rel);
    }

    for (const sym of result.symbols) {
      symbolTable.add(sym.filePath, sym.name, sym.nodeId, sym.type, {
        parameterCount: sym.parameterCount,
        requiredParameterCount: sym.requiredParameterCount,
        parameterTypes: sym.parameterTypes,
        returnType: sym.returnType,
        declaredType: sym.declaredType,
        ownerId: sym.ownerId,
        qualifiedName: sym.qualifiedName,
      });
    }

    for (const item of result.imports) allImports.push(item);
    for (const item of result.calls) allCalls.push(item);
    for (const item of result.assignments) allAssignments.push(item);
    for (const item of result.heritage) allHeritage.push(item);
    for (const item of result.routes) allRoutes.push(item);
    for (const item of result.fetchCalls) allFetchCalls.push(item);
    for (const item of result.decoratorRoutes) allDecoratorRoutes.push(item);
    for (const item of result.toolDefs) allToolDefs.push(item);
    if (result.ormQueries) for (const item of result.ormQueries) allORMQueries.push(item);
    for (const item of result.constructorBindings) allConstructorBindings.push(item);
    if (result.fileScopeBindings) {
      for (const item of result.fileScopeBindings) fileScopeBindingsByFile.push(item);
    }
    if (result.parsedFiles) {
      for (const item of result.parsedFiles) allParsedFiles.push(item);
    }
  }

  const skippedLanguages = new Map<string, number>();
  for (const result of chunkResults) {
    for (const [lang, count] of Object.entries(result.skippedLanguages)) {
      skippedLanguages.set(lang, (skippedLanguages.get(lang) || 0) + count);
    }
  }
  if (skippedLanguages.size > 0) {
    const summary = Array.from(skippedLanguages.entries())
      .map(([lang, count]) => `${lang}: ${count}`)
      .join(', ');
    console.warn(`  Skipped unsupported languages: ${summary}`);
  }

  onFileProgress?.(total, total, 'done');
  return {
    imports: allImports,
    calls: allCalls,
    assignments: allAssignments,
    heritage: allHeritage,
    routes: allRoutes,
    fetchCalls: allFetchCalls,
    decoratorRoutes: allDecoratorRoutes,
    toolDefs: allToolDefs,
    ormQueries: allORMQueries,
    constructorBindings: allConstructorBindings,
    fileScopeBindings: fileScopeBindingsByFile,
    parsedFiles: allParsedFiles,
  };
};
