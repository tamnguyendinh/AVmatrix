const LOCAL_RUNTIME_ONLY_MESSAGE =
  'The legacy prompt context builder has been retired. Use the local session runtime bridge instead.';

export interface CodebaseStats {
  projectName: string;
  fileCount: number;
  functionCount: number;
  classCount: number;
  interfaceCount: number;
  methodCount: number;
}

export interface Hotspot {
  name: string;
  type: string;
  filePath: string;
  connections: number;
}

export interface CodebaseContext {
  stats: CodebaseStats;
  hotspots: Hotspot[];
  folderTree: string;
}

const throwLocalRuntimeOnly = (): never => {
  throw new Error(LOCAL_RUNTIME_ONLY_MESSAGE);
};

export async function getCodebaseStats(
  _executeQuery: (cypher: string) => Promise<any[]>,
  _projectName: string,
): Promise<never> {
  return throwLocalRuntimeOnly();
}

export async function getHotspots(
  _executeQuery: (cypher: string) => Promise<any[]>,
): Promise<never> {
  return throwLocalRuntimeOnly();
}

export async function getFolderTree(
  _executeQuery: (cypher: string) => Promise<any[]>,
): Promise<never> {
  return throwLocalRuntimeOnly();
}

export async function buildCodebaseContext(
  _executeQuery: (cypher: string) => Promise<any[]>,
  _projectName: string,
): Promise<never> {
  return throwLocalRuntimeOnly();
}

export function formatContextForPrompt(_context: CodebaseContext): never {
  return throwLocalRuntimeOnly();
}

export function buildDynamicSystemPrompt(_basePrompt: string, _context: CodebaseContext): never {
  return throwLocalRuntimeOnly();
}
