import type { AgentStreamChunk } from './types.local-runtime';
import type { CodebaseContext } from './context-builder.retired-local-runtime';
import type { GraphRAGBackend } from './tools.retired-local-runtime';

const LOCAL_RUNTIME_ONLY_MESSAGE =
  'The legacy web agent path has been retired. Use the local session runtime bridge instead.';

export const BASE_SYSTEM_PROMPT =
  'GitNexus web uses the local session runtime bridge. Legacy web agent paths are retired.';

export interface AgentMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

const throwLocalRuntimeOnly = (): never => {
  throw new Error(LOCAL_RUNTIME_ONLY_MESSAGE);
};

export const createChatModel = (_config: unknown): never => throwLocalRuntimeOnly();

export const createGraphRAGAgent = (
  _config: unknown,
  _backend: GraphRAGBackend,
  _codebaseContext?: CodebaseContext,
): never => throwLocalRuntimeOnly();

export async function* streamAgentResponse(
  _agent: unknown,
  _messages: AgentMessage[],
): AsyncGenerator<AgentStreamChunk, void, void> {
  throwLocalRuntimeOnly();
  yield undefined as never;
}

export const invokeAgent = async (_agent: unknown, _messages: AgentMessage[]): Promise<never> =>
  throwLocalRuntimeOnly();
