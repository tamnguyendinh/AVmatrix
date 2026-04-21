import type { EnrichedSearchResult, GrepResult } from '../../services/backend-client';

const LOCAL_RUNTIME_ONLY_MESSAGE =
  'The legacy Graph RAG tool factory has been retired. Use the local session runtime bridge instead.';

export interface GraphRAGBackend {
  executeQuery: (cypher: string) => Promise<Record<string, unknown>[]>;
  search: (
    query: string,
    opts?: { limit?: number; mode?: 'hybrid' | 'semantic' | 'bm25'; enrich?: boolean },
  ) => Promise<EnrichedSearchResult[]>;
  grep: (pattern: string, limit?: number) => Promise<GrepResult[]>;
  readFile: (filePath: string) => Promise<string>;
}

export const createGraphRAGTools = (_backend: GraphRAGBackend): never => {
  throw new Error(LOCAL_RUNTIME_ONLY_MESSAGE);
};
