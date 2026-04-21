/**
 * Local-runtime-first LLM/runtime types.
 *
 * The active product path uses the shared local session runtime. Legacy
 * provider identifiers remain only as compatibility literals so older stored
 * settings can be normalized back to Codex during migration.
 */

export type LegacyLLMProvider =
  | 'openai'
  | 'azure-openai'
  | 'gemini'
  | 'anthropic'
  | 'ollama'
  | 'openrouter'
  | 'minimax'
  | 'glm';

export type LLMProvider = 'codex' | LegacyLLMProvider;

export interface CodexConfig {
  provider: 'codex';
  model: string;
  temperature?: number;
  maxTokens?: number;
}

export type ProviderConfig = CodexConfig;

/**
 * Persisted settings for the local runtime path.
 *
 * `activeProvider` is pinned to `codex`. Compatibility helpers may accept
 * legacy provider names at the API edge, but storage is always normalized back
 * to this shape.
 */
export interface LLMSettings {
  activeProvider: 'codex';
  codex?: Partial<Omit<CodexConfig, 'provider'>>;
  intelligentClustering: boolean;
  hasSeenClusteringPrompt: boolean;
  useSameModelForClustering: boolean;
}

export const DEFAULT_LLM_SETTINGS: LLMSettings = {
  activeProvider: 'codex',
  intelligentClustering: false,
  hasSeenClusteringPrompt: false,
  useSameModelForClustering: true,
  codex: {
    model: 'codex-account',
    temperature: 0,
  },
};

export interface MessageStep {
  id: string;
  type: 'reasoning' | 'tool_call' | 'content';
  content?: string;
  toolCall?: ToolCallInfo;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'tool';
  content: string;
  /** @deprecated Use steps instead for proper ordering */
  toolCalls?: ToolCallInfo[];
  steps?: MessageStep[];
  toolCallId?: string;
  timestamp: number;
}

export interface ToolCallInfo {
  id: string;
  name: string;
  args: Record<string, unknown>;
  result?: string;
  status: 'pending' | 'running' | 'completed' | 'error';
}

export interface AgentStreamChunk {
  type: 'reasoning' | 'tool_call' | 'tool_result' | 'content' | 'error' | 'done';
  reasoning?: string;
  content?: string;
  toolCall?: ToolCallInfo;
  error?: string;
}

export interface AgentStep {
  id: string;
  type: 'reasoning' | 'tool_call' | 'answer';
  content?: string;
  toolCall?: ToolCallInfo;
  timestamp: number;
}

/**
 * Retained only to preserve the old export surface for compatibility imports.
 * The local session runtime no longer injects this prompt schema into the web
 * agent path.
 */
export const GRAPH_SCHEMA_DESCRIPTION =
  'AVmatrix web uses the local session runtime bridge; provider-based graph prompt injection is retired.';
